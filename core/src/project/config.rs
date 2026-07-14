use std::collections::HashMap;
use std::env;
use std::error::Error;
use std::fmt;
use std::fs;
use std::io;
use std::path::{Path, PathBuf};

use sha2::{Digest, Sha256};

use super::{MavenEnvironment, ProjectEnvironment, ProjectType, is_maven_version};
use crate::java::JdkInstallation;

const PROJECTS_DIRECTORY_NAME: &str = "projects";
const ENVIRONMENT_FILE_EXTENSION: &str = "properties";

/// Error raised while loading or saving a project environment.
#[derive(Debug)]
#[non_exhaustive]
pub enum ProjectConfigError {
    NotFound {
        start: PathBuf,
    },
    Read {
        path: PathBuf,
        source: io::Error,
    },
    Write {
        path: PathBuf,
        source: io::Error,
    },
    ResolveProjectPath {
        path: PathBuf,
        source: io::Error,
    },
    StorageDirectoryUnavailable,
    InvalidLine {
        path: PathBuf,
        line: usize,
    },
    DuplicateKey {
        path: PathBuf,
        key: String,
    },
    MissingKey {
        path: PathBuf,
        key: &'static str,
    },
    InvalidValue {
        path: PathBuf,
        key: &'static str,
        value: String,
    },
}

impl fmt::Display for ProjectConfigError {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::NotFound { start } => write!(
                formatter,
                "could not find a saved javaup environment for {} or its parent directories",
                start.display()
            ),
            Self::Read { path, source } => {
                write!(formatter, "could not read {}: {source}", path.display())
            }
            Self::Write { path, source } => {
                write!(formatter, "could not write {}: {source}", path.display())
            }
            Self::ResolveProjectPath { path, source } => write!(
                formatter,
                "could not resolve project path {}: {source}",
                path.display()
            ),
            Self::StorageDirectoryUnavailable => write!(
                formatter,
                "could not determine the javaup storage directory; set JAVAUP_HOME"
            ),
            Self::InvalidLine { path, line } => write!(
                formatter,
                "invalid environment entry at {}:{line}; expected key=value",
                path.display()
            ),
            Self::DuplicateKey { path, key } => {
                write!(formatter, "duplicate key '{key}' in {}", path.display())
            }
            Self::MissingKey { path, key } => {
                write!(formatter, "missing key '{key}' in {}", path.display())
            }
            Self::InvalidValue { path, key, value } => write!(
                formatter,
                "invalid value '{value}' for key '{key}' in {}",
                path.display()
            ),
        }
    }
}

impl Error for ProjectConfigError {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        match self {
            Self::Read { source, .. }
            | Self::Write { source, .. }
            | Self::ResolveProjectPath { source, .. } => Some(source),
            _ => None,
        }
    }
}

pub(super) fn save(
    project_dir: &Path,
    environment: &ProjectEnvironment,
) -> Result<PathBuf, ProjectConfigError> {
    save_in(&storage_directory()?, project_dir, environment)
}

fn save_in(
    storage_directory: &Path,
    project_dir: &Path,
    environment: &ProjectEnvironment,
) -> Result<PathBuf, ProjectConfigError> {
    let path = environment_path(storage_directory, project_dir)?;
    let absolute_project_dir = std::path::absolute(project_dir).map_err(|source| {
        ProjectConfigError::ResolveProjectPath {
            path: project_dir.to_owned(),
            source,
        }
    })?;
    let contents = format!(
        "project.path={}\nproject.type={}\njava.version={}\njava.home={}\nmaven.version={}\nmaven.wrapper={}\n",
        absolute_project_dir.display(),
        environment.project_type.as_str(),
        environment.java.major_version(),
        environment.java.home().display(),
        environment.maven.version,
        environment.maven.uses_wrapper
    );
    fs::create_dir_all(path.parent().expect("environment path must have a parent")).map_err(
        |source| ProjectConfigError::Write {
            path: path.clone(),
            source,
        },
    )?;
    fs::write(&path, contents).map_err(|source| ProjectConfigError::Write {
        path: path.clone(),
        source,
    })?;
    Ok(path)
}

pub(super) fn load(project_dir: &Path) -> Result<ProjectEnvironment, ProjectConfigError> {
    load_in(&storage_directory()?, project_dir)
}

fn load_in(
    storage_directory: &Path,
    project_dir: &Path,
) -> Result<ProjectEnvironment, ProjectConfigError> {
    let path = environment_path(storage_directory, project_dir)?;
    load_path(&path)
}

fn load_path(path: &Path) -> Result<ProjectEnvironment, ProjectConfigError> {
    let contents = fs::read_to_string(path).map_err(|source| ProjectConfigError::Read {
        path: path.to_owned(),
        source,
    })?;
    let values = parse_entries(path, &contents)?;

    let project_type = required(path, &values, "project.type")?;
    if project_type != "maven" {
        return Err(invalid_value(path, "project.type", project_type));
    }

    let java_value = required(path, &values, "java.version")?;
    let java_version = java_value
        .parse::<u32>()
        .ok()
        .filter(|version| *version >= 5)
        .ok_or_else(|| invalid_value(path, "java.version", java_value))?;

    let java_home_value = required(path, &values, "java.home")?;
    let java_home = PathBuf::from(java_home_value);
    if !java_home.is_absolute() {
        return Err(invalid_value(path, "java.home", java_home_value));
    }

    let maven_version = required(path, &values, "maven.version")?;
    if !is_maven_version(maven_version) {
        return Err(invalid_value(path, "maven.version", maven_version));
    }

    let wrapper_value = required(path, &values, "maven.wrapper")?;
    let uses_wrapper = wrapper_value
        .parse::<bool>()
        .map_err(|_| invalid_value(path, "maven.wrapper", wrapper_value))?;

    Ok(ProjectEnvironment {
        project_type: ProjectType::Maven,
        java: JdkInstallation::recorded(java_version, java_home),
        maven: MavenEnvironment {
            version: maven_version.to_owned(),
            uses_wrapper,
        },
    })
}

pub(super) fn load_nearest(
    start: &Path,
) -> Result<(PathBuf, ProjectEnvironment), ProjectConfigError> {
    load_nearest_in(&storage_directory()?, start)
}

fn load_nearest_in(
    storage_directory: &Path,
    start: &Path,
) -> Result<(PathBuf, ProjectEnvironment), ProjectConfigError> {
    for directory in start.ancestors() {
        let path = environment_path(storage_directory, directory)?;
        if path.is_file() {
            return load_path(&path).map(|environment| (directory.to_owned(), environment));
        }
    }

    Err(ProjectConfigError::NotFound {
        start: start.to_owned(),
    })
}

fn environment_path(
    storage_directory: &Path,
    project_dir: &Path,
) -> Result<PathBuf, ProjectConfigError> {
    let canonical_project_dir =
        fs::canonicalize(project_dir).map_err(|source| ProjectConfigError::ResolveProjectPath {
            path: project_dir.to_owned(),
            source,
        })?;
    let identity = project_identity(&canonical_project_dir);
    let digest = Sha256::digest(identity.as_bytes());
    let project_key = encode_hex(&digest);
    Ok(storage_directory
        .join(PROJECTS_DIRECTORY_NAME)
        .join(format!("{project_key}.{ENVIRONMENT_FILE_EXTENSION}")))
}

fn encode_hex(bytes: &[u8]) -> String {
    const HEX_DIGITS: &[u8; 16] = b"0123456789abcdef";

    let mut encoded = String::with_capacity(bytes.len() * 2);
    for byte in bytes {
        encoded.push(HEX_DIGITS[(byte >> 4) as usize] as char);
        encoded.push(HEX_DIGITS[(byte & 0x0f) as usize] as char);
    }
    encoded
}

#[cfg(windows)]
fn project_identity(path: &Path) -> String {
    path.to_string_lossy().to_lowercase()
}

#[cfg(not(windows))]
fn project_identity(path: &Path) -> String {
    path.to_string_lossy().into_owned()
}

fn storage_directory() -> Result<PathBuf, ProjectConfigError> {
    if let Some(path) = env::var_os("JAVAUP_HOME").filter(|path| !path.is_empty()) {
        return Ok(PathBuf::from(path));
    }

    platform_storage_directory().ok_or(ProjectConfigError::StorageDirectoryUnavailable)
}

#[cfg(windows)]
fn platform_storage_directory() -> Option<PathBuf> {
    env::var_os("APPDATA")
        .filter(|path| !path.is_empty())
        .map(PathBuf::from)
        .map(|path| path.join(crate::PRODUCT_NAME))
}

#[cfg(target_os = "macos")]
fn platform_storage_directory() -> Option<PathBuf> {
    env::var_os("HOME")
        .filter(|path| !path.is_empty())
        .map(PathBuf::from)
        .map(|path| {
            path.join("Library")
                .join("Application Support")
                .join(crate::PRODUCT_NAME)
        })
}

#[cfg(all(unix, not(target_os = "macos")))]
fn platform_storage_directory() -> Option<PathBuf> {
    if let Some(path) = env::var_os("XDG_CONFIG_HOME")
        .filter(|path| !path.is_empty())
        .map(PathBuf::from)
        .filter(|path| path.is_absolute())
    {
        return Some(path.join(crate::PRODUCT_NAME));
    }

    env::var_os("HOME")
        .filter(|path| !path.is_empty())
        .map(PathBuf::from)
        .map(|path| path.join(".config").join(crate::PRODUCT_NAME))
}

#[cfg(not(any(windows, unix)))]
fn platform_storage_directory() -> Option<PathBuf> {
    None
}

fn parse_entries(
    path: &Path,
    contents: &str,
) -> Result<HashMap<String, String>, ProjectConfigError> {
    let mut values = HashMap::new();
    for (index, line) in contents.lines().enumerate() {
        let line = line.trim();
        if line.is_empty() || line.starts_with('#') {
            continue;
        }
        let (key, value) = line
            .split_once('=')
            .ok_or_else(|| ProjectConfigError::InvalidLine {
                path: path.to_owned(),
                line: index + 1,
            })?;
        let key = key.trim();
        let value = value.trim();
        if key.is_empty() {
            return Err(ProjectConfigError::InvalidLine {
                path: path.to_owned(),
                line: index + 1,
            });
        }
        if values.insert(key.to_owned(), value.to_owned()).is_some() {
            return Err(ProjectConfigError::DuplicateKey {
                path: path.to_owned(),
                key: key.to_owned(),
            });
        }
    }
    Ok(values)
}

fn required<'a>(
    path: &Path,
    values: &'a HashMap<String, String>,
    key: &'static str,
) -> Result<&'a str, ProjectConfigError> {
    values
        .get(key)
        .map(String::as_str)
        .filter(|value| !value.is_empty())
        .ok_or_else(|| ProjectConfigError::MissingKey {
            path: path.to_owned(),
            key,
        })
}

fn invalid_value(path: &Path, key: &'static str, value: &str) -> ProjectConfigError {
    ProjectConfigError::InvalidValue {
        path: path.to_owned(),
        key,
        value: value.to_owned(),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn environment(project_dir: &Path, java_version: u32) -> ProjectEnvironment {
        ProjectEnvironment {
            project_type: ProjectType::Maven,
            java: JdkInstallation::recorded(
                java_version,
                project_dir.join(format!("jdk-{java_version}")),
            ),
            maven: MavenEnvironment {
                version: "3.9.9".to_owned(),
                uses_wrapper: true,
            },
        }
    }

    #[test]
    fn saves_and_loads_an_environment_outside_the_project() {
        let project = tempfile::tempdir().unwrap();
        let storage = tempfile::tempdir().unwrap();
        let environment = environment(project.path(), 17);

        let path = save_in(storage.path(), project.path(), &environment).unwrap();
        let loaded = load_in(storage.path(), project.path()).unwrap();

        assert_eq!(
            path.parent().unwrap(),
            storage.path().join(PROJECTS_DIRECTORY_NAME)
        );
        assert_eq!(
            path.extension().and_then(|value| value.to_str()),
            Some("properties")
        );
        let project_key = path.file_stem().unwrap().to_string_lossy();
        assert_eq!(project_key.len(), 64);
        assert!(
            project_key
                .chars()
                .all(|character| character.is_ascii_hexdigit())
        );
        assert_eq!(fs::read_dir(project.path()).unwrap().count(), 0);
        assert_eq!(loaded, environment);
        assert_eq!(
            fs::read_to_string(path).unwrap(),
            format!(
                "project.path={}\nproject.type=maven\njava.version=17\njava.home={}\nmaven.version=3.9.9\nmaven.wrapper=true\n",
                project.path().display(),
                project.path().join("jdk-17").display()
            )
        );
    }

    #[test]
    fn rejects_incomplete_and_invalid_files() {
        let project = tempfile::tempdir().unwrap();
        let storage = tempfile::tempdir().unwrap();
        let path = environment_path(storage.path(), project.path()).unwrap();
        fs::create_dir_all(path.parent().unwrap()).unwrap();
        fs::write(
            &path,
            "project.type=maven\njava.version=abc\njava.home=/jdk\nmaven.version=3.9.9\nmaven.wrapper=false\n",
        )
        .unwrap();

        let error = load_in(storage.path(), project.path()).unwrap_err();
        assert!(matches!(
            error,
            ProjectConfigError::InvalidValue {
                key: "java.version",
                ..
            }
        ));
    }

    #[test]
    fn loads_the_nearest_parent_environment_from_user_storage() {
        let project = tempfile::tempdir().unwrap();
        let storage = tempfile::tempdir().unwrap();
        let child = project.path().join("module").join("src");
        fs::create_dir_all(&child).unwrap();
        save_in(
            storage.path(),
            project.path(),
            &environment(project.path(), 17),
        )
        .unwrap();

        let (project_dir, environment) = load_nearest_in(storage.path(), &child).unwrap();

        assert_eq!(project_dir, project.path());
        assert_eq!(environment.java_version(), 17);
    }

    #[test]
    fn uses_distinct_storage_files_for_distinct_projects() {
        let parent = tempfile::tempdir().unwrap();
        let storage = tempfile::tempdir().unwrap();
        let first = parent.path().join("first");
        let second = parent.path().join("second");
        fs::create_dir_all(&first).unwrap();
        fs::create_dir_all(&second).unwrap();

        assert_ne!(
            environment_path(storage.path(), &first).unwrap(),
            environment_path(storage.path(), &second).unwrap()
        );
    }
}
