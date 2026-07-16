use std::env;
use std::ffi::OsString;
use std::io::{self, Write};
use std::process::ExitStatus;

use javaup_core::project::ProjectEnvironment;

use crate::output::Output;

pub(super) fn execute<Stdout, Stderr>(
    maven_arguments: Vec<OsString>,
    output: &mut Output<'_, Stdout, Stderr>,
) -> io::Result<ExitStatus>
where
    Stdout: Write,
    Stderr: Write,
{
    let current_dir = env::current_dir()?;
    let (project_dir, environment) =
        ProjectEnvironment::load_nearest(&current_dir).map_err(io::Error::other)?;
    let java_version = environment
        .installed_java_version()
        .map_err(io::Error::other)?;
    let maven_source = if environment.maven().uses_wrapper() {
        "Maven Wrapper"
    } else {
        "PATH"
    };
    let command_line = format_command(&maven_arguments);

    output.info(format_args!("Starting Maven command: {command_line}"))?;
    output.info(format_args!(
        "Environment: JDK {java_version} at {}; Maven {} from {maven_source}; directory {}",
        environment.java_home().display(),
        environment.maven().version(),
        current_dir.display()
    ))?;

    let mut command = environment
        .maven_command(&project_dir, maven_arguments)
        .map_err(io::Error::other)?;
    let status = command.current_dir(current_dir).status()?;

    if status.success() {
        output.success(format_args!(
            "Completed Maven command: {command_line} (JDK {java_version}; Maven {} from {maven_source})",
            environment.maven().version()
        ))?;
    } else {
        output.error(format_args!(
            "Maven command failed with {}: {command_line} (JDK {java_version}; Maven {} from {maven_source})",
            exit_description(status),
            environment.maven().version()
        ))?;
    }

    Ok(status)
}

fn format_command(arguments: &[OsString]) -> String {
    let mut command = String::from("mvn");
    for argument in arguments {
        command.push(' ');
        let argument = argument.to_string_lossy();
        if argument.is_empty() || argument.chars().any(char::is_whitespace) {
            command.push_str(&format!("{argument:?}"));
        } else {
            command.push_str(&argument);
        }
    }
    command
}

fn exit_description(status: ExitStatus) -> String {
    status
        .code()
        .map(|code| format!("exit code {code}"))
        .unwrap_or_else(|| "process termination".to_owned())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn formats_maven_arguments_without_adding_defaults() {
        assert_eq!(format_command(&[]), "mvn");
        assert_eq!(
            format_command(&[
                OsString::from("verify"),
                OsString::from("-DskipTests"),
                OsString::from("-Dmessage=hello world"),
            ]),
            "mvn verify -DskipTests \"-Dmessage=hello world\""
        );
    }
}
