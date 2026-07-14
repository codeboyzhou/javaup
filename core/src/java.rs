//! Installed JDK discovery and validation.

use std::collections::HashSet;
use std::env;
use std::error::Error;
use std::ffi::{OsStr, OsString};
use std::fmt;
use std::fs;
use std::io;
use std::path::{Path, PathBuf};
use std::process::{Command, ExitStatus};

/// A validated JDK installation.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct JdkInstallation {
    major_version: u32,
    home: PathBuf,
}

impl JdkInstallation {
    /// Discovers an installed JDK matching `major_version`.
    pub fn discover(major_version: u32) -> Result<Self, JdkDiscoveryError> {
        let override_variable = override_variable(major_version);
        if let Some(home) = env::var_os(&override_variable) {
            return inspect_home(PathBuf::from(home), major_version).map_err(|source| {
                JdkDiscoveryError::InvalidOverride {
                    variable: override_variable,
                    source,
                }
            });
        }

        for home in candidate_homes(major_version) {
            if let Ok(installation) = inspect_home(home, major_version) {
                return Ok(installation);
            }
        }

        Err(JdkDiscoveryError::NotFound {
            major_version,
            override_variable,
        })
    }

    /// Re-validates that the recorded home still contains the expected JDK.
    pub fn validate(&self) -> Result<(), JdkValidationError> {
        inspect_home(self.home.clone(), self.major_version).map(|_| ())
    }

    pub fn major_version(&self) -> u32 {
        self.major_version
    }

    pub fn home(&self) -> &Path {
        &self.home
    }

    pub(crate) fn recorded(major_version: u32, home: PathBuf) -> Self {
        Self {
            major_version,
            home,
        }
    }
}

/// Error raised while discovering a JDK installation.
#[derive(Debug)]
#[non_exhaustive]
pub enum JdkDiscoveryError {
    InvalidOverride {
        variable: String,
        source: JdkValidationError,
    },
    NotFound {
        major_version: u32,
        override_variable: String,
    },
}

impl fmt::Display for JdkDiscoveryError {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::InvalidOverride { variable, source } => {
                write!(formatter, "invalid JDK configured by {variable}: {source}")
            }
            Self::NotFound {
                major_version,
                override_variable,
            } => write!(
                formatter,
                "could not find an installed JDK {major_version}; set {override_variable} to its home directory"
            ),
        }
    }
}

impl Error for JdkDiscoveryError {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        match self {
            Self::InvalidOverride { source, .. } => Some(source),
            Self::NotFound { .. } => None,
        }
    }
}

/// Error raised when a recorded JDK home is no longer usable.
#[derive(Debug)]
#[non_exhaustive]
pub enum JdkValidationError {
    HomeNotAbsolute {
        home: PathBuf,
    },
    HomeNotFound {
        home: PathBuf,
    },
    JavaNotFound {
        path: PathBuf,
    },
    CompilerNotFound {
        path: PathBuf,
    },
    VersionCommand {
        path: PathBuf,
        source: io::Error,
    },
    VersionCommandFailed {
        path: PathBuf,
        status: ExitStatus,
    },
    VersionNotFound {
        home: PathBuf,
    },
    VersionMismatch {
        home: PathBuf,
        expected: u32,
        actual: u32,
    },
}

impl fmt::Display for JdkValidationError {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::HomeNotAbsolute { home } => {
                write!(formatter, "{} is not an absolute path", home.display())
            }
            Self::HomeNotFound { home } => {
                write!(formatter, "{} is not a directory", home.display())
            }
            Self::JavaNotFound { path } => {
                write!(
                    formatter,
                    "Java launcher was not found at {}",
                    path.display()
                )
            }
            Self::CompilerNotFound { path } => {
                write!(
                    formatter,
                    "Java compiler was not found at {}",
                    path.display()
                )
            }
            Self::VersionCommand { path, source } => {
                write!(
                    formatter,
                    "could not run '{} -version': {source}",
                    path.display()
                )
            }
            Self::VersionCommandFailed { path, status } => {
                write!(
                    formatter,
                    "'{} -version' failed with status {status}",
                    path.display()
                )
            }
            Self::VersionNotFound { home } => write!(
                formatter,
                "could not determine the Java version in {}",
                home.display()
            ),
            Self::VersionMismatch {
                home,
                expected,
                actual,
            } => write!(
                formatter,
                "{} contains JDK {actual}, but JDK {expected} is required",
                home.display()
            ),
        }
    }
}

impl Error for JdkValidationError {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        match self {
            Self::VersionCommand { source, .. } => Some(source),
            _ => None,
        }
    }
}

fn inspect_home(home: PathBuf, expected_major: u32) -> Result<JdkInstallation, JdkValidationError> {
    if !home.is_absolute() {
        return Err(JdkValidationError::HomeNotAbsolute { home });
    }
    if !home.is_dir() {
        return Err(JdkValidationError::HomeNotFound { home });
    }

    let java = home.join("bin").join(executable_name("java"));
    if !java.is_file() {
        return Err(JdkValidationError::JavaNotFound { path: java });
    }

    let compiler = home.join("bin").join(executable_name("javac"));
    if !compiler.is_file() {
        return Err(JdkValidationError::CompilerNotFound { path: compiler });
    }

    let major_version = match version_from_release_file(&home) {
        Some(version) => Some(version),
        None => version_from_command(&java)?,
    }
    .ok_or_else(|| JdkValidationError::VersionNotFound { home: home.clone() })?;

    if major_version != expected_major {
        return Err(JdkValidationError::VersionMismatch {
            home,
            expected: expected_major,
            actual: major_version,
        });
    }

    Ok(JdkInstallation {
        major_version,
        home,
    })
}

fn version_from_release_file(home: &Path) -> Option<u32> {
    let contents = fs::read_to_string(home.join("release")).ok()?;
    contents.lines().find_map(|line| {
        let (key, value) = line.split_once('=')?;
        (key.trim() == "JAVA_VERSION")
            .then(|| value.trim().trim_matches(['\"', '\'']))
            .and_then(parse_java_major)
    })
}

fn version_from_command(java: &Path) -> Result<Option<u32>, JdkValidationError> {
    let output = Command::new(java)
        .arg("-version")
        .output()
        .map_err(|source| JdkValidationError::VersionCommand {
            path: java.to_owned(),
            source,
        })?;
    if !output.status.success() {
        return Err(JdkValidationError::VersionCommandFailed {
            path: java.to_owned(),
            status: output.status,
        });
    }

    let combined = format!(
        "{}\n{}",
        String::from_utf8_lossy(&output.stdout),
        String::from_utf8_lossy(&output.stderr)
    );
    Ok(parse_java_version_output(&combined))
}

fn parse_java_version_output(output: &str) -> Option<u32> {
    output.lines().find_map(|line| {
        let (_, version) = line.split_once("version")?;
        let version = version.trim();
        let version = version
            .strip_prefix('\"')
            .and_then(|value| value.split_once('\"').map(|(value, _)| value))
            .unwrap_or_else(|| version.split_whitespace().next().unwrap_or(version));
        parse_java_major(version)
    })
}

pub(crate) fn parse_java_major(value: &str) -> Option<u32> {
    let value = value.trim().trim_start_matches(['\"', '\'', '[', '(']);
    let major = value.strip_prefix("1.").unwrap_or(value);
    let digits: String = major
        .chars()
        .take_while(|character| character.is_ascii_digit())
        .collect();
    let major = digits.parse().ok()?;
    (major >= 5).then_some(major)
}

fn override_variable(major_version: u32) -> String {
    format!("JAVAUP_JDK_{major_version}_HOME")
}

fn candidate_homes(major_version: u32) -> Vec<PathBuf> {
    let mut candidates = CandidateHomes::default();
    let mut nearby_roots = CandidateHomes::default();

    for variable in [
        format!("JAVA_HOME_{major_version}"),
        format!("JAVA{major_version}_HOME"),
        format!("JDK_HOME_{major_version}"),
        format!("JDK{major_version}_HOME"),
        "JAVA_HOME".to_owned(),
        "JDK_HOME".to_owned(),
    ] {
        if let Some(home) = env::var_os(variable) {
            push_home_and_parent(&mut candidates, &mut nearby_roots, home);
        }
    }

    if let Some(path) = env::var_os("PATH") {
        for directory in env::split_paths(&path) {
            for executable in [executable_name("java"), executable_name("javac")] {
                let path = directory.join(executable);
                let Ok(path) = fs::canonicalize(path) else {
                    continue;
                };
                let Some(home) = path.parent().and_then(Path::parent) else {
                    continue;
                };
                if home.join("release").is_file() {
                    push_home_and_parent(&mut candidates, &mut nearby_roots, home);
                }
            }
        }
    }

    for root in nearby_roots.values.into_iter().chain(installation_roots()) {
        push_homes_under_root(&mut candidates, &root);
    }

    candidates.values
}

fn push_home_and_parent(
    candidates: &mut CandidateHomes,
    nearby_roots: &mut CandidateHomes,
    home: impl AsRef<OsStr>,
) {
    let home = PathBuf::from(home.as_ref());
    candidates.push(&home);
    if let Some(parent) = home.parent().filter(|parent| parent.is_absolute()) {
        nearby_roots.push(parent);
    }
}

fn push_homes_under_root(candidates: &mut CandidateHomes, root: &Path) {
    candidates.push(root.join("current"));
    let Ok(entries) = fs::read_dir(root) else {
        return;
    };
    let mut entries: Vec<_> = entries.flatten().map(|entry| entry.path()).collect();
    entries.sort();
    for entry in entries {
        candidates.push(&entry);
        candidates.push(entry.join("current"));
        candidates.push(entry.join("Contents").join("Home"));
    }
}

#[derive(Default)]
struct CandidateHomes {
    seen: HashSet<OsString>,
    values: Vec<PathBuf>,
}

impl CandidateHomes {
    fn push(&mut self, path: impl AsRef<OsStr>) {
        let path = PathBuf::from(path.as_ref());
        if path.as_os_str().is_empty() {
            return;
        }
        let key = path.as_os_str().to_os_string();
        if self.seen.insert(key) {
            self.values.push(path);
        }
    }
}

fn installation_roots() -> Vec<PathBuf> {
    let mut roots = Vec::new();

    #[cfg(windows)]
    {
        for variable in ["ProgramFiles", "ProgramFiles(x86)", "ProgramW6432"] {
            if let Some(path) = env::var_os(variable) {
                let path = PathBuf::from(path);
                roots.extend([
                    path.join("Java"),
                    path.join("Eclipse Adoptium"),
                    path.join("Microsoft"),
                    path.join("Amazon Corretto"),
                    path.join("Zulu"),
                ]);
            }
        }
        if let Some(path) = env::var_os("LOCALAPPDATA") {
            roots.push(
                PathBuf::from(path)
                    .join("Programs")
                    .join("Eclipse Adoptium"),
            );
        }
    }

    #[cfg(not(windows))]
    roots.extend([
        PathBuf::from("/usr/lib/jvm"),
        PathBuf::from("/usr/java"),
        PathBuf::from("/opt/java"),
        PathBuf::from("/opt/jdk"),
    ]);

    #[cfg(target_os = "macos")]
    roots.push(PathBuf::from("/Library/Java/JavaVirtualMachines"));

    if let Some(home) = home_directory() {
        roots.extend([
            home.join(".jdks"),
            home.join(".jabba").join("jdk"),
            home.join(".sdkman").join("candidates").join("java"),
            home.join("scoop").join("apps"),
        ]);
        #[cfg(target_os = "macos")]
        roots.push(
            home.join("Library")
                .join("Java")
                .join("JavaVirtualMachines"),
        );
    }

    roots
}

fn home_directory() -> Option<PathBuf> {
    env::var_os(if cfg!(windows) { "USERPROFILE" } else { "HOME" }).map(PathBuf::from)
}

fn executable_name(name: &str) -> OsString {
    if cfg!(windows) {
        OsString::from(format!("{name}.exe"))
    } else {
        OsString::from(name)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn fake_jdk(major_version: &str) -> tempfile::TempDir {
        let directory = tempfile::tempdir().unwrap();
        write_fake_jdk(directory.path(), major_version);
        directory
    }

    fn write_fake_jdk(home: &Path, major_version: &str) {
        let bin = home.join("bin");
        fs::create_dir_all(&bin).unwrap();
        fs::write(
            home.join("release"),
            format!("JAVA_VERSION=\"{major_version}\"\n"),
        )
        .unwrap();
        fs::write(bin.join(executable_name("java")), "").unwrap();
        fs::write(bin.join(executable_name("javac")), "").unwrap();
    }

    #[test]
    fn parses_legacy_and_modern_java_versions() {
        assert_eq!(parse_java_major("1.8.0_442"), Some(8));
        assert_eq!(parse_java_major("17.0.12"), Some(17));
        assert_eq!(parse_java_major("[21,)"), Some(21));
        assert_eq!(parse_java_major("invalid"), None);
        assert_eq!(
            parse_java_version_output("openjdk version \"17.0.12\" 2024-07-16"),
            Some(17)
        );
    }

    #[test]
    fn validates_a_jdk_from_its_release_file() {
        let directory = fake_jdk("1.8.0_442");
        let installation = inspect_home(directory.path().to_owned(), 8).unwrap();

        assert_eq!(installation.major_version(), 8);
        assert_eq!(installation.home(), directory.path());
    }

    #[test]
    fn rejects_a_jdk_with_the_wrong_major_version() {
        let directory = fake_jdk("17.0.12");
        let error = inspect_home(directory.path().to_owned(), 8).unwrap_err();

        assert!(matches!(
            error,
            JdkValidationError::VersionMismatch {
                expected: 8,
                actual: 17,
                ..
            }
        ));
    }

    #[test]
    fn discovers_a_jdk_beside_a_configured_jdk_home() {
        let directory = tempfile::tempdir().unwrap();
        let configured_home = directory.path().join("OpenJDK17");
        let expected_home = directory.path().join("OpenJDK8");
        write_fake_jdk(&configured_home, "17.0.12");
        write_fake_jdk(&expected_home, "1.8.0_442");

        let mut candidates = CandidateHomes::default();
        let mut nearby_roots = CandidateHomes::default();
        push_home_and_parent(&mut candidates, &mut nearby_roots, &configured_home);
        for root in nearby_roots.values {
            push_homes_under_root(&mut candidates, &root);
        }
        let installation = candidates
            .values
            .into_iter()
            .find_map(|home| inspect_home(home, 8).ok())
            .unwrap();

        assert_eq!(installation.home(), expected_home);
    }
}
