use std::path::Path;
use std::process::ExitCode;

use buildopt_hermetic_helper::{CapabilityProbe, parse_manifest, render_enforcement};

fn main() -> ExitCode {
    match run(std::env::args().skip(1).collect()) {
        Ok(code) => ExitCode::from(code),
        Err((code, message)) => {
            eprintln!("buildopt-hermetic-helper: {message}");
            ExitCode::from(code)
        }
    }
}

fn run(arguments: Vec<String>) -> Result<u8, (u8, String)> {
    match arguments.as_slice() {
        [command] if command == "probe" => {
            print!("{}", CapabilityProbe::run().render_json());
            Ok(0)
        }
        [command, manifest_flag, manifest, separator, producer, ..]
            if command == "enforce"
                && manifest_flag == "--manifest"
                && separator == "--" =>
        {
            let parsed = parse_manifest(Path::new(manifest), Path::new(producer))
                .map_err(|message| (65, message))?;
            let probe = CapabilityProbe::run();
            print!("{}", render_enforcement(&parsed, &probe));
            Ok(78)
        }
        _ => Err((
            64,
            "usage: buildopt-hermetic-helper probe | enforce --manifest <path> -- <command> [args...]"
                .to_owned(),
        )),
    }
}
