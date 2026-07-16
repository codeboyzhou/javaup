use std::env;
use std::ffi::OsString;
use std::fs;
use std::io;
use std::path::{Path, PathBuf};
use std::process::{Command, Output};

use super::is_maven_version;

const WRAPPER_PROPERTIES_PATH: &str = ".mvn/wrapper/maven-wrapper.properties";

pub(super) fn wrapper_properties_path(project_dir: &Path) -> PathBuf {
    project_dir.join(WRAPPER_PROPERTIES_PATH)
}

pub(super) fn find_wrapper(project_dir: &Path) -> Option<PathBuf> {
    wrapper_names()
        .iter()
        .map(|name| project_dir.join(name))
        .find(|path| is_usable_executable(path))
}

pub(super) fn find_maven_on_path() -> Option<PathBuf> {
    let path = env::var_os("PATH")?;
    for directory in env::split_paths(&path) {
        for name in maven_command_names() {
            let candidate = directory.join(name);
            if is_usable_executable(&candidate) {
                return Some(candidate);
            }
        }
    }
    None
}

pub(super) fn run_maven_version(
    executable: &Path,
    project_dir: &Path,
    java_home: &Path,
) -> Result<Output, MavenProbeError> {
    let path = path_with_java(java_home).map_err(MavenProbeError::InvalidPath)?;
    Command::new(executable)
        .arg("--version")
        .current_dir(project_dir)
        .env("JAVA_HOME", java_home)
        .env("PATH", path)
        .output()
        .map_err(MavenProbeError::Command)
}

pub(super) fn configure_java(
    command: &mut Command,
    java_home: &Path,
) -> Result<(), env::JoinPathsError> {
    command
        .env("JAVA_HOME", java_home)
        .env("PATH", path_with_java(java_home)?);
    Ok(())
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

#[cfg(unix)]
fn is_usable_executable(path: &Path) -> bool {
    use std::os::unix::fs::PermissionsExt;

    fs::metadata(path)
        .is_ok_and(|metadata| metadata.is_file() && metadata.permissions().mode() & 0o111 != 0)
}

#[cfg(not(unix))]
fn is_usable_executable(path: &Path) -> bool {
    path.is_file()
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
    use std::ffi::OsStr;

    use super::*;

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
        let mut command = Command::new(OsStr::new("mvn"));
        let java_home = Path::new("jdk");
        let result = configure_java(&mut command, java_home);
        assert!(result.is_ok());
    }
}
