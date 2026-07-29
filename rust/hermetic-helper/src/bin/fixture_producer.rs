#![forbid(unsafe_code)]

use std::fs;
use std::io::Read;
use std::net::{SocketAddr, TcpStream};
use std::path::Path;
use std::process::{Command, ExitCode};
use std::time::{Duration, SystemTime};

fn main() -> ExitCode {
    let arguments = std::env::args().skip(1).collect::<Vec<_>>();
    let [input, output] = arguments.as_slice() else {
        eprintln!("usage: hermetic-fixture-producer <input> <output>");
        return ExitCode::from(64);
    };
    match run(Path::new(input), Path::new(output)) {
        Ok(()) => ExitCode::SUCCESS,
        Err(message) => {
            eprintln!("hermetic-fixture-producer: {message}");
            ExitCode::FAILURE
        }
    }
}

fn run(input: &Path, output: &Path) -> Result<(), String> {
    fs::read_to_string(input).map_err(|error| format!("filesystem input: {error}"))?;
    let _environment_present = std::env::var_os("PATH").is_some();

    let child = Command::new("/bin/sh")
        .args(["-c", "exit 0"])
        .status()
        .map_err(|error| format!("native child: {error}"))?;
    if !child.success() {
        return Err("native child failed".to_owned());
    }

    let address = "127.0.0.1:9"
        .parse::<SocketAddr>()
        .map_err(|error| error.to_string())?;
    let _network_attempt = TcpStream::connect_timeout(&address, Duration::from_millis(100));
    let _clock = SystemTime::now();
    let mut entropy = [0_u8; 16];
    fs::File::open("/dev/urandom")
        .and_then(|mut file| file.read_exact(&mut entropy))
        .map_err(|error| format!("randomness: {error}"))?;

    if let Some(parent) = output.parent() {
        fs::create_dir_all(parent).map_err(|error| format!("output directory: {error}"))?;
    }
    fs::write(
        output,
        concat!(
            "FILESYSTEM=EXECUTED\n",
            "PROCESS_TREE=EXECUTED\n",
            "NATIVE_CHILD=EXECUTED\n",
            "NETWORK=EXECUTED\n",
            "ENVIRONMENT=EXECUTED\n",
            "CLOCK=EXECUTED\n",
            "RANDOMNESS=EXECUTED\n"
        ),
    )
    .map_err(|error| format!("output: {error}"))
}
