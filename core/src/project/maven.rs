use std::env;
use std::ffi::OsString;
use std::fs;
use std::io;
use std::path::{Path, PathBuf};

use super::is_maven_version;
use crate::executable;
use crate::process::{ProcessInvocation, ProcessOutput, ProcessRunner};

const WRAPPER_PROPERTIES_PATH: &str = ".mvn/wrapper/maven-wrapper.properties";

pub(super) fn wrapper_properties_path(project_dir: &Path) -> PathBuf {
    project_dir.join(WRAPPER_PROPERTIES_PATH)
}

pub(super) fn find_wrapper(project_dir: &Path) -> Option<PathBuf> {
    wrapper_names()
        .iter()
        .map(|name| project_dir.join(name))
        .find(|path| executable::is_usable(path))
}

pub(super) fn find_maven_on_path() -> Option<PathBuf> {
    let path = env::var_os("PATH")?;
    for directory in env::split_paths(&path) {
        for name in maven_command_names() {
            let candidate = directory.join(name);
            if executable::is_usable(&candidate) {
                return Some(candidate);
            }
        }
    }
    None
}

pub(super) fn run_maven_version<R>(
    runner: &R,
    executable: &Path,
    project_dir: &Path,
    java_home: &Path,
) -> Result<ProcessOutput, MavenProbeError>
where
    R: ProcessRunner + ?Sized,
{
    let invocation = invocation(executable, project_dir, java_home, ["--version".into()])
        .map_err(MavenProbeError::InvalidPath)?;
    runner.output(&invocation).map_err(MavenProbeError::Command)
}

pub(super) fn invocation(
    executable: &Path,
    project_dir: &Path,
    java_home: &Path,
    arguments: impl IntoIterator<Item = OsString>,
) -> Result<ProcessInvocation, env::JoinPathsError> {
    let mut invocation =
        ProcessInvocation::new(executable.to_owned(), arguments, project_dir.to_owned());
    invocation.set_env("JAVA_HOME", java_home.as_os_str());
    invocation.set_env("PATH", path_with_java(java_home)?);
    Ok(invocation)
}

fn path_with_java(java_home: &Path) -> Result<OsString, env::JoinPathsError> {
    let mut paths = vec![java_home.join("bin")];
    if let Some(path) = env::var_os("PATH") {
        paths.extend(env::split_paths(&path));
    }
    env::join_paths(paths)
}

pub(super) fn read_wrapper_version(properties_path: &Path) -> io::Result<Option<String>> {
    fs::read_to_string(properties_path).map(|contents| parse_wrapper_maven_version(&contents))
}

pub(super) fn parse_wrapper_maven_version(contents: &str) -> Option<String> {
    let distribution_url = contents.lines().find_map(|line| {
        let line = line.trim();
        if line.is_empty() || line.starts_with('#') || line.starts_with('!') {
            return None;
        }
        let (key, value) = split_property(line)?;
        (key == "distributionUrl").then(|| value.replace("\\:", ":"))
    })?;

    extract_version_after_marker(&distribution_url, "apache-maven-")
        .or_else(|| extract_version_after_marker(&distribution_url, "maven-mvnd-"))
}

fn split_property(line: &str) -> Option<(&str, &str)> {
    let separator = line
        .char_indices()
        .find(|(_, character)| matches!(character, '=' | ':') || character.is_whitespace())?
        .0;
    let (key, remainder) = line.split_at(separator);
    let value = remainder
        .trim_start_matches(|character: char| {
            character == '=' || character == ':' || character.is_whitespace()
        })
        .trim();
    Some((key.trim(), value))
}

fn extract_version_after_marker(value: &str, marker: &str) -> Option<String> {
    let remainder = value.rsplit_once(marker)?.1;
    let archive_name = remainder.split(['?', '#']).next()?.trim();
    let version = ["-bin.zip", "-bin.tar.gz", ".zip", ".tar.gz"]
        .into_iter()
        .find_map(|suffix| archive_name.strip_suffix(suffix))?;
    is_maven_version(version).then(|| version.to_owned())
}

pub(super) fn parse_maven_version_output(output: &str) -> Option<String> {
    output.lines().find_map(|line| {
        let line = strip_ansi_codes(line);
        let remainder = line.trim().strip_prefix("Apache Maven ")?;
        let version = remainder.split_whitespace().next()?;
        is_maven_version(version).then(|| version.to_owned())
    })
}

fn strip_ansi_codes(value: &str) -> String {
    let mut result = String::with_capacity(value.len());
    let mut chars = value.chars().peekable();
    while let Some(character) = chars.next() {
        if character == '\u{1b}' && chars.peek() == Some(&'[') {
            chars.next();
            for sequence_character in chars.by_ref() {
                if sequence_character.is_ascii_alphabetic() {
                    break;
                }
            }
        } else {
            result.push(character);
        }
    }
    result
}

#[cfg(windows)]
fn wrapper_names() -> &'static [&'static str] {
    &["mvnw.cmd", "mvnw.bat", "mvnw.exe", "mvnw"]
}

#[cfg(not(windows))]
fn wrapper_names() -> &'static [&'static str] {
    &["mvnw"]
}

#[cfg(windows)]
fn maven_command_names() -> &'static [&'static str] {
    &["mvn.cmd", "mvn.bat", "mvn.exe", "mvn"]
}

#[cfg(not(windows))]
fn maven_command_names() -> &'static [&'static str] {
    &["mvn"]
}

#[derive(Debug)]
pub(super) enum MavenProbeError {
    Command(io::Error),
    InvalidPath(env::JoinPathsError),
}

impl MavenProbeError {
    pub(super) fn command_source(self) -> io::Error {
        match self {
            Self::Command(source) => source,
            Self::InvalidPath(source) => io::Error::new(io::ErrorKind::InvalidInput, source),
        }
    }
}

#[cfg(test)]
mod tests {
    use std::cell::RefCell;
    use std::ffi::OsStr;

    use super::*;

    struct RecordingRunner {
        invocations: RefCell<Vec<ProcessInvocation>>,
    }

    impl RecordingRunner {
        fn new() -> Self {
            Self {
                invocations: RefCell::new(Vec::new()),
            }
        }
    }

    impl ProcessRunner for RecordingRunner {
        fn output(&self, invocation: &ProcessInvocation) -> io::Result<ProcessOutput> {
            self.invocations.borrow_mut().push(invocation.clone());
            Ok(ProcessOutput::new(
                successful_exit_status(),
                b"Apache Maven 3.9.9\n".to_vec(),
                Vec::new(),
            ))
        }
    }

    #[cfg(unix)]
    fn successful_exit_status() -> std::process::ExitStatus {
        use std::os::unix::process::ExitStatusExt;

        std::process::ExitStatus::from_raw(0)
    }

    #[cfg(windows)]
    fn successful_exit_status() -> std::process::ExitStatus {
        use std::os::windows::process::ExitStatusExt;

        std::process::ExitStatus::from_raw(0)
    }

    #[test]
    fn parses_supported_property_separators_and_maven_output() {
        assert_eq!(
            parse_wrapper_maven_version(
                "distributionUrl: https\\://example.test/apache-maven-3.9.9-bin.zip"
            ),
            Some("3.9.9".to_owned())
        );
        assert_eq!(
            parse_maven_version_output("\u{1b}[1mApache Maven 4.0.0-rc-4\u{1b}[m\n"),
            Some("4.0.0-rc-4".to_owned())
        );
    }

    #[test]
    fn rejects_non_maven_distribution_urls() {
        assert_eq!(
            parse_wrapper_maven_version("distributionUrl=https://example.test/tool.zip"),
            None
        );
    }

    #[test]
    fn command_names_are_not_empty() {
        assert!(!maven_command_names().is_empty());
        assert!(!wrapper_names().is_empty());
    }

    #[test]
    fn os_strings_are_preserved_when_configuring_java() {
        let java_home = Path::new("jdk");
        let invocation = invocation(
            Path::new("mvn"),
            Path::new("project"),
            java_home,
            [OsString::from("verify")],
        )
        .unwrap();
        assert_eq!(invocation.program(), Path::new("mvn"));
        assert_eq!(
            invocation.arguments().collect::<Vec<_>>(),
            [OsStr::new("verify")]
        );
        assert_eq!(
            invocation
                .environment()
                .find(|(key, _)| *key == OsStr::new("JAVA_HOME"))
                .map(|(_, value)| value),
            Some(java_home.as_os_str())
        );
    }

    #[test]
    fn probing_uses_the_injected_runner_and_selected_jdk() {
        let runner = RecordingRunner::new();
        let output = run_maven_version(
            &runner,
            Path::new("mvn"),
            Path::new("project"),
            Path::new("jdk-21"),
        )
        .unwrap();

        assert!(output.status().success());
        let invocations = runner.invocations.borrow();
        assert_eq!(invocations.len(), 1);
        let invocation = &invocations[0];
        assert_eq!(invocation.program(), Path::new("mvn"));
        assert_eq!(
            invocation.arguments().collect::<Vec<_>>(),
            [OsStr::new("--version")]
        );
        assert_eq!(
            invocation
                .environment()
                .find(|(key, _)| *key == OsStr::new("JAVA_HOME"))
                .map(|(_, value)| value),
            Some(OsStr::new("jdk-21"))
        );
    }
}
