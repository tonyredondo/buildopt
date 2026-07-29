#![forbid(unsafe_code)]

use std::collections::BTreeMap;
use std::fs;
use std::os::unix::fs::PermissionsExt;
use std::path::{Path, PathBuf};
use std::process::Command;

pub const REPORT_SCHEMA: &str = "buildopt.spikes/hermetic-helper-report/v1";
pub const MANIFEST_SCHEMA: &str = "buildopt.spikes/hermetic-producer-manifest/v1";

#[derive(Debug, Clone, Eq, PartialEq)]
pub struct ProducerManifest {
    pub task_execution_id: String,
    pub producer_id: String,
    pub workspace: PathBuf,
    pub input: PathBuf,
    pub output: PathBuf,
    pub temporary: PathBuf,
    pub command: PathBuf,
}

#[derive(Debug, Clone, Eq, PartialEq)]
pub struct CapabilityProbe {
    pub user_namespace: bool,
    pub mount_namespace: bool,
    pub pid_namespace: bool,
    pub network_namespace: bool,
    pub seccomp_kernel: bool,
    pub landlock_interface: bool,
    pub cgroup_delegated: bool,
}

impl CapabilityProbe {
    #[must_use]
    pub fn run() -> Self {
        Self {
            user_namespace: unshare(&["--user", "--map-root-user"]),
            mount_namespace: unshare(&["--user", "--map-root-user", "--mount"]),
            pid_namespace: unshare(&["--user", "--map-root-user", "--pid", "--fork"]),
            network_namespace: unshare(&["--user", "--map-root-user", "--net"]),
            seccomp_kernel: fs::read_to_string("/proc/sys/kernel/seccomp/actions_avail")
                .is_ok_and(|actions| actions.split_whitespace().any(|item| item == "allow")),
            landlock_interface: Path::new("/sys/kernel/security/landlock").exists(),
            cgroup_delegated: command_success("test", &["-w", "/sys/fs/cgroup"]),
        }
    }

    #[must_use]
    pub fn trace_complete(&self) -> bool {
        false
    }

    #[must_use]
    pub fn render_json(&self) -> String {
        format!(
            concat!(
                "{{\n",
                "  \"schemaVersion\": \"{}\",\n",
                "  \"mode\": \"ENFORCE_ISOLATED\",\n",
                "  \"traceComplete\": false,\n",
                "  \"qualification\": \"UNAVAILABLE\",\n",
                "  \"pendingPublication\": \"ABORTED\",\n",
                "  \"fallback\": \"DISCARD_CANDIDATE_AND_RUN_BASELINE\",\n",
                "  \"reason\": \"UNMEDIATED_CLOCK_RANDOMNESS_AND_POLICY_GAPS\",\n",
                "  \"namespaces\": {{\n",
                "    \"user\": {},\n",
                "    \"mount\": {},\n",
                "    \"pid\": {},\n",
                "    \"network\": {}\n",
                "  }},\n",
                "  \"kernel\": {{\n",
                "    \"seccompAdvertised\": {},\n",
                "    \"landlockInterfaceVisible\": {},\n",
                "    \"cgroupDelegated\": {}\n",
                "  }},\n",
                "  \"coverage\": {{\n",
                "    \"filesystem\": \"NAMESPACE_PROBED_NOT_ENFORCED\",\n",
                "    \"processTree\": \"PID_NAMESPACE_PROBED_NOT_ENFORCED\",\n",
                "    \"network\": \"NETWORK_NAMESPACE_PROBED_NOT_ENFORCED\",\n",
                "    \"environment\": \"UNMEDIATED\",\n",
                "    \"clock\": \"UNMEDIATED_VDSO\",\n",
                "    \"randomness\": \"UNMEDIATED_GETRANDOM\"\n",
                "  }}\n",
                "}}\n"
            ),
            REPORT_SCHEMA,
            self.user_namespace,
            self.mount_namespace,
            self.pid_namespace,
            self.network_namespace,
            self.seccomp_kernel,
            self.landlock_interface,
            self.cgroup_delegated
        )
    }
}

pub fn parse_manifest(path: &Path, requested_command: &Path) -> Result<ProducerManifest, String> {
    let content = fs::read_to_string(path).map_err(|error| format!("read manifest: {error}"))?;
    let mut fields = BTreeMap::new();
    for (line_index, line) in content.lines().enumerate() {
        if line.is_empty() || line.starts_with('#') {
            continue;
        }
        let (key, value) = line
            .split_once('=')
            .ok_or_else(|| format!("line {} is not key=value", line_index + 1))?;
        if key.is_empty() || value.is_empty() || fields.insert(key, value).is_some() {
            return Err(format!(
                "line {} has an empty or duplicate field",
                line_index + 1
            ));
        }
    }
    let expected = [
        "schemaVersion",
        "producerMode",
        "taskExecutionId",
        "producerId",
        "workspace",
        "input",
        "output",
        "temporary",
        "command",
        "network",
        "clock",
        "randomness",
    ];
    if fields.keys().copied().collect::<Vec<_>>() != {
        let mut sorted = expected;
        sorted.sort_unstable();
        sorted.to_vec()
    } {
        return Err("manifest fields do not match the closed schema".to_owned());
    }
    require(&fields, "schemaVersion", MANIFEST_SCHEMA)?;
    require(&fields, "producerMode", "DEDICATED_TASK")?;
    require(&fields, "network", "DENY")?;
    require(&fields, "clock", "DENY")?;
    require(&fields, "randomness", "DENY")?;

    let task_execution_id = non_empty_identifier(&fields, "taskExecutionId")?;
    let producer_id = non_empty_identifier(&fields, "producerId")?;
    let workspace = canonical_directory(field(&fields, "workspace")?, "workspace")?;
    let input = canonical_path(field(&fields, "input")?, "input")?;
    let output = canonical_directory(field(&fields, "output")?, "output")?;
    let temporary = canonical_directory(field(&fields, "temporary")?, "temporary")?;
    let command = canonical_path(field(&fields, "command")?, "command")?;
    let requested = canonical_path(
        requested_command
            .to_str()
            .ok_or_else(|| "requested command is not UTF-8".to_owned())?,
        "requested command",
    )?;

    require_child(&workspace, &input, "input")?;
    require_child(&workspace, &output, "output")?;
    require_child(&workspace, &temporary, "temporary")?;
    require_child(&workspace, &command, "command")?;
    if command != requested {
        return Err("requested command does not match the manifest".to_owned());
    }
    if input.starts_with(&output) || input.starts_with(&temporary) {
        return Err("input overlaps a writable directory".to_owned());
    }
    if output.starts_with(&temporary) || temporary.starts_with(&output) {
        return Err("output and temporary directories overlap".to_owned());
    }
    if input
        .metadata()
        .map_err(|error| error.to_string())?
        .permissions()
        .mode()
        & 0o222
        != 0
    {
        return Err("declared input must be read-only".to_owned());
    }
    if !command.is_file() {
        return Err("producer command must be a regular file".to_owned());
    }

    Ok(ProducerManifest {
        task_execution_id,
        producer_id,
        workspace,
        input,
        output,
        temporary,
        command,
    })
}

#[must_use]
pub fn render_enforcement(manifest: &ProducerManifest, probe: &CapabilityProbe) -> String {
    format!(
        concat!(
            "{{\n",
            "  \"schemaVersion\": \"{}\",\n",
            "  \"taskExecutionId\": \"{}\",\n",
            "  \"producerId\": \"{}\",\n",
            "  \"producerMode\": \"DEDICATED_TASK\",\n",
            "  \"traceComplete\": {},\n",
            "  \"qualification\": \"UNAVAILABLE\",\n",
            "  \"candidateExecuted\": false,\n",
            "  \"candidateDiscarded\": true,\n",
            "  \"pendingPublication\": \"ABORTED\",\n",
            "  \"fallback\": \"DISCARD_CANDIDATE_AND_RUN_BASELINE\",\n",
            "  \"reason\": \"UNMEDIATED_CLOCK_RANDOMNESS_AND_POLICY_GAPS\"\n",
            "}}\n"
        ),
        REPORT_SCHEMA,
        json_escape(&manifest.task_execution_id),
        json_escape(&manifest.producer_id),
        probe.trace_complete()
    )
}

fn unshare(arguments: &[&str]) -> bool {
    let mut command = Command::new("unshare");
    command.args(arguments).arg("/bin/true");
    command.status().is_ok_and(|status| status.success())
}

fn command_success(program: &str, arguments: &[&str]) -> bool {
    Command::new(program)
        .args(arguments)
        .status()
        .is_ok_and(|status| status.success())
}

fn require(fields: &BTreeMap<&str, &str>, key: &str, expected: &str) -> Result<(), String> {
    if field(fields, key)? != expected {
        return Err(format!("{key} must equal {expected}"));
    }
    Ok(())
}

fn field<'a>(fields: &'a BTreeMap<&str, &str>, key: &str) -> Result<&'a str, String> {
    fields
        .get(key)
        .copied()
        .ok_or_else(|| format!("missing field {key}"))
}

fn non_empty_identifier(fields: &BTreeMap<&str, &str>, key: &str) -> Result<String, String> {
    let value = field(fields, key)?;
    if value.len() > 128
        || !value
            .bytes()
            .all(|byte| byte.is_ascii_alphanumeric() || matches!(byte, b'-' | b'_' | b'.'))
    {
        return Err(format!("{key} is not a bounded identifier"));
    }
    Ok(value.to_owned())
}

fn canonical_directory(value: &str, name: &str) -> Result<PathBuf, String> {
    let path = canonical_path(value, name)?;
    if !path.is_dir() {
        return Err(format!("{name} must be a directory"));
    }
    Ok(path)
}

fn canonical_path(value: &str, name: &str) -> Result<PathBuf, String> {
    let path = Path::new(value);
    if !path.is_absolute() {
        return Err(format!("{name} must be absolute"));
    }
    fs::canonicalize(path).map_err(|error| format!("canonicalize {name}: {error}"))
}

fn require_child(workspace: &Path, path: &Path, name: &str) -> Result<(), String> {
    if path == workspace || !path.starts_with(workspace) {
        return Err(format!("{name} must be a strict workspace child"));
    }
    Ok(())
}

fn json_escape(value: &str) -> String {
    value.replace('\\', "\\\\").replace('"', "\\\"")
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::os::unix::fs::PermissionsExt;
    use std::sync::atomic::{AtomicUsize, Ordering};

    static NEXT: AtomicUsize = AtomicUsize::new(1);

    #[test]
    fn parses_exact_task_specific_manifest() {
        let fixture = Fixture::new();
        let manifest = parse_manifest(&fixture.manifest, &fixture.command).expect("valid manifest");
        assert_eq!(manifest.task_execution_id, "task-17");
        assert_eq!(manifest.producer_id, "producer-9");
    }

    #[test]
    fn rejects_writable_input() {
        let fixture = Fixture::new();
        fs::set_permissions(&fixture.input, fs::Permissions::from_mode(0o644))
            .expect("make input writable");
        let failure = parse_manifest(&fixture.manifest, &fixture.command)
            .expect_err("writable input must fail");
        assert!(failure.contains("read-only"));
    }

    #[test]
    fn rejects_command_outside_workspace() {
        let fixture = Fixture::new();
        let outside = std::env::current_exe().expect("current executable");
        let content = fs::read_to_string(&fixture.manifest)
            .expect("read manifest")
            .replace(
                fixture.command.to_str().expect("command path"),
                outside.to_str().expect("outside path"),
            );
        fs::write(&fixture.manifest, content).expect("replace command");
        let failure =
            parse_manifest(&fixture.manifest, &outside).expect_err("outside command must fail");
        assert!(failure.contains("strict workspace child"));
    }

    #[test]
    fn incomplete_probe_never_executes_candidate() {
        let fixture = Fixture::new();
        let manifest = parse_manifest(&fixture.manifest, &fixture.command).expect("valid manifest");
        let probe = CapabilityProbe {
            user_namespace: true,
            mount_namespace: true,
            pid_namespace: true,
            network_namespace: true,
            seccomp_kernel: true,
            landlock_interface: false,
            cgroup_delegated: false,
        };
        let report = render_enforcement(&manifest, &probe);
        assert!(report.contains("\"candidateExecuted\": false"));
        assert!(report.contains("\"pendingPublication\": \"ABORTED\""));
    }

    struct Fixture {
        root: PathBuf,
        input: PathBuf,
        command: PathBuf,
        manifest: PathBuf,
    }

    impl Fixture {
        fn new() -> Self {
            let root = std::env::temp_dir().join(format!(
                "buildopt-hermetic-test-{}-{}",
                std::process::id(),
                NEXT.fetch_add(1, Ordering::Relaxed)
            ));
            let input = root.join("inputs/source.txt");
            let output = root.join("outputs");
            let temporary = root.join("tmp");
            let command = root.join("producer/run");
            fs::create_dir_all(input.parent().expect("input parent")).expect("inputs");
            fs::create_dir_all(&output).expect("outputs");
            fs::create_dir_all(&temporary).expect("temporary");
            fs::create_dir_all(command.parent().expect("command parent")).expect("producer");
            fs::write(&input, "input").expect("input");
            fs::set_permissions(&input, fs::Permissions::from_mode(0o444)).expect("read-only");
            fs::write(&command, "#!/bin/sh\nexit 0\n").expect("command");
            fs::set_permissions(&command, fs::Permissions::from_mode(0o755)).expect("executable");
            let manifest = root.join("manifest.properties");
            fs::write(
                &manifest,
                format!(
                    concat!(
                        "schemaVersion={}\n",
                        "producerMode=DEDICATED_TASK\n",
                        "taskExecutionId=task-17\n",
                        "producerId=producer-9\n",
                        "workspace={}\n",
                        "input={}\n",
                        "output={}\n",
                        "temporary={}\n",
                        "command={}\n",
                        "network=DENY\n",
                        "clock=DENY\n",
                        "randomness=DENY\n"
                    ),
                    MANIFEST_SCHEMA,
                    root.display(),
                    input.display(),
                    output.display(),
                    temporary.display(),
                    command.display()
                ),
            )
            .expect("manifest");
            Self {
                root,
                input,
                command,
                manifest,
            }
        }
    }

    impl Drop for Fixture {
        fn drop(&mut self) {
            let _ = fs::set_permissions(&self.input, fs::Permissions::from_mode(0o644));
            let _ = fs::remove_dir_all(&self.root);
        }
    }
}
