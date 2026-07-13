use std::env;
use std::ffi::OsString;
use std::io::{self, Write};
use std::process::ExitStatus;

use javaup_core::project::ProjectEnvironment;

pub(super) fn execute<W>(
    mut maven_arguments: Vec<OsString>,
    stdout: &mut W,
) -> io::Result<ExitStatus>
where
    W: Write,
{
    let current_dir = env::current_dir()?;
    let (project_dir, environment) =
        ProjectEnvironment::load_nearest(&current_dir).map_err(io::Error::other)?;

    if maven_arguments.is_empty() {
        maven_arguments.extend([OsString::from("clean"), OsString::from("package")]);
    }

    writeln!(
        stdout,
        "Using JDK {} from {} (Maven {}, wrapper: {})",
        environment.java_version(),
        environment.java_home().display(),
        environment.maven().version(),
        environment.maven().uses_wrapper()
    )?;
    stdout.flush()?;

    environment
        .maven_command(&project_dir, &maven_arguments)
        .map_err(io::Error::other)?
        .status()
}
