use std::env;
use std::ffi::OsString;
use std::io::{self, Write};
use std::process::ExitStatus;

use javaup_core::maven_settings::MavenSettingsProfile;
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
    let explicit_settings = find_explicit_settings(&maven_arguments);
    let settings_profile_name = environment.maven().settings_profile();
    if explicit_settings.is_some()
        && let Some(profile_name) = settings_profile_name
    {
        output.warning(format_args!(
            "Command-line Maven settings override project settings '{profile_name}'"
        ))?;
    }

    let settings_selection = match explicit_settings {
        Some(settings) => SettingsSelection::CommandLine(settings),
        None => match settings_profile_name {
            Some(name) => SettingsSelection::Project(
                MavenSettingsProfile::resolve(name).map_err(io::Error::other)?,
            ),
            None => SettingsSelection::Default,
        },
    };
    let settings_summary = settings_selection.summary();
    let maven_arguments = settings_selection.apply(maven_arguments);
    let command_line = format_command(&maven_arguments);

    output.info(format_args!("Starting Maven command: {command_line}"))?;
    output.info(format_args!(
        "Environment: JDK {java_version} at {}; Maven {} from {maven_source}; directory {}",
        environment.java_home().display(),
        environment.maven().version(),
        current_dir.display()
    ))?;
    output.info(format_args!("Maven settings: {settings_summary}"))?;

    let mut command = environment
        .maven_command(&project_dir, maven_arguments)
        .map_err(io::Error::other)?;
    let status = command.current_dir(current_dir).status()?;

    if status.success() {
        output.success(format_args!(
            "Completed Maven command: {command_line} (JDK {java_version}; Maven {} from {maven_source}; settings {settings_summary})",
            environment.maven().version(),
        ))?;
    } else {
        output.error(format_args!(
            "Maven command failed with {}: {command_line} (JDK {java_version}; Maven {} from {maven_source}; settings {settings_summary})",
            exit_description(status),
            environment.maven().version(),
        ))?;
    }

    Ok(status)
}

#[derive(Debug, Eq, PartialEq)]
struct ExplicitSettings {
    path: Option<OsString>,
}

enum SettingsSelection {
    Default,
    Project(MavenSettingsProfile),
    CommandLine(ExplicitSettings),
}

impl SettingsSelection {
    fn apply(&self, arguments: Vec<OsString>) -> Vec<OsString> {
        let Self::Project(profile) = self else {
            return arguments;
        };

        let mut effective_arguments = Vec::with_capacity(arguments.len() + 2);
        effective_arguments.push(OsString::from("--settings"));
        effective_arguments.push(profile.path().as_os_str().to_owned());
        effective_arguments.extend(arguments);
        effective_arguments
    }

    fn summary(&self) -> String {
        match self {
            Self::Default => "Maven default".to_owned(),
            Self::Project(profile) => format!(
                "profile '{}' ({})",
                profile.name(),
                profile.path().display()
            ),
            Self::CommandLine(settings) => settings
                .path
                .as_deref()
                .map(|path| format!("command line ({})", path.to_string_lossy()))
                .unwrap_or_else(|| "command line (missing path)".to_owned()),
        }
    }
}

fn find_explicit_settings(arguments: &[OsString]) -> Option<ExplicitSettings> {
    for (index, argument) in arguments.iter().enumerate() {
        let argument_text = argument.to_string_lossy();
        if argument_text == "-s" || argument_text == "--settings" {
            return Some(ExplicitSettings {
                path: arguments.get(index + 1).cloned(),
            });
        }
        if let Some(path) = argument_text
            .strip_prefix("--settings=")
            .or_else(|| argument_text.strip_prefix("-s="))
            .filter(|path| !path.is_empty())
        {
            return Some(ExplicitSettings {
                path: Some(OsString::from(path)),
            });
        }
        if let Some(path) = argument_text
            .strip_prefix("-s")
            .filter(|path| !path.is_empty())
        {
            return Some(ExplicitSettings {
                path: Some(OsString::from(path)),
            });
        }
    }
    None
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

    #[test]
    fn detects_alternate_settings_arguments() {
        assert_eq!(
            find_explicit_settings(&["-s".into(), "nexus.xml".into(), "verify".into()]),
            Some(ExplicitSettings {
                path: Some("nexus.xml".into())
            })
        );
        assert_eq!(
            find_explicit_settings(&["--settings=google.xml".into()]),
            Some(ExplicitSettings {
                path: Some("google.xml".into())
            })
        );
        assert_eq!(find_explicit_settings(&["verify".into()]), None);
    }
}
