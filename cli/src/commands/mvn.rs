use std::env;
use std::ffi::OsString;
use std::io::Write;
use std::process::ExitStatus;

use javaup_core::maven_settings::MavenSettingsProfile;
use javaup_core::project::ProjectEnvironment;

use super::CommandError;
use crate::output::Output;

pub(super) fn execute<Stdout, Stderr>(
    maven_arguments: Vec<OsString>,
    output: &mut Output<'_, Stdout, Stderr>,
) -> Result<ExitStatus, CommandError>
where
    Stdout: Write,
    Stderr: Write,
{
    let current_dir = env::current_dir()?;
    let (project_dir, environment) = ProjectEnvironment::load_nearest(&current_dir)?;
    let java_version = environment.installed_java_version()?;
    let maven = environment
        .maven()
        .ok_or(CommandError::UnsupportedBuildTool {
            expected: "maven",
            actual: environment.build_tool().as_str(),
        })?;
    let maven_source = if maven.uses_wrapper() {
        "Maven Wrapper"
    } else {
        "PATH"
    };
    let explicit_settings = find_explicit_settings(&maven_arguments);
    let settings_profile_name = maven.settings_profile();
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
            Some(name) => SettingsSelection::Project(MavenSettingsProfile::resolve(name)?),
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
        maven.version(),
        current_dir.display()
    ))?;
    output.info(format_args!("Maven settings: {settings_summary}"))?;

    let invocation = environment
        .maven_invocation(&project_dir, maven_arguments)?
        .with_current_dir(current_dir);
    let status = crate::process::status(&invocation)?;

    if status.success() {
        output.success(format_args!(
            "Completed Maven command: {command_line} (JDK {java_version}; Maven {} from {maven_source}; settings {settings_summary})",
            maven.version(),
        ))?;
    } else {
        output.error(format_args!(
            "Maven command failed with {}: {command_line} (JDK {java_version}; Maven {} from {maven_source}; settings {settings_summary})",
            exit_description(status),
            maven.version(),
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
    let mut redact_next = false;
    for argument in arguments {
        command.push(' ');
        let argument = argument.to_string_lossy();
        let (argument, should_redact_next) = if redact_next {
            ("<redacted>".to_owned(), false)
        } else {
            redact_argument(&argument)
        };
        redact_next = should_redact_next;
        if argument.is_empty() || argument.chars().any(char::is_whitespace) {
            command.push_str(&format!("{argument:?}"));
        } else {
            command.push_str(&argument);
        }
    }
    command
}

fn redact_argument(argument: &str) -> (String, bool) {
    if let Some(property) = argument.strip_prefix("-D")
        && let Some((key, _)) = property.split_once('=')
        && is_sensitive_key(key)
    {
        return (format!("-D{key}=<redacted>"), false);
    }

    if let Some((key, _)) = argument.split_once('=')
        && key.starts_with('-')
        && is_sensitive_key(key.trim_start_matches('-'))
    {
        return (format!("{key}=<redacted>"), false);
    }

    let key = argument.trim_start_matches('-');
    if argument.starts_with('-') && is_sensitive_key(key) {
        return (argument.to_owned(), true);
    }
    (argument.to_owned(), false)
}

fn is_sensitive_key(key: &str) -> bool {
    let normalized = key.to_ascii_lowercase().replace(['-', '_'], ".");
    normalized.split('.').any(|component| {
        matches!(
            component,
            "password"
                | "passwd"
                | "passphrase"
                | "token"
                | "secret"
                | "apikey"
                | "credential"
                | "credentials"
                | "privatekey"
                | "accesskey"
        )
    })
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
    fn redacts_secrets_from_command_summaries() {
        assert_eq!(
            format_command(&[
                "deploy".into(),
                "-Drepository.password=hunter2".into(),
                "--token".into(),
                "top-secret".into(),
                "-DskipTests=true".into(),
            ]),
            "mvn deploy -Drepository.password=<redacted> --token <redacted> -DskipTests=true"
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
