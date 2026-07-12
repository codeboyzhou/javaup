use std::collections::HashMap;
use std::error::Error;
use std::fmt;
use std::fs;
use std::io;
use std::path::{Path, PathBuf};

use super::{
    ENVIRONMENT_FILE_NAME, MavenEnvironment, ProjectEnvironment, ProjectType, is_maven_version,
};

/// Error raised while loading or saving a `.javaup` environment file.
#[derive(Debug)]
#[non_exhaustive]
pub enum ProjectConfigError {
    Read {
        path: PathBuf,
        source: io::Error,
    },
    Write {
        path: PathBuf,
        source: io::Error,
    },
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
            Self::Read { path, source } => {
                write!(formatter, "could not read {}: {source}", path.display())
            }
            Self::Write { path, source } => {
                write!(formatter, "could not write {}: {source}", path.display())
            }
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
            Self::Read { source, .. } | Self::Write { source, .. } => Some(source),
            _ => None,
        }
    }
}

pub(super) fn save(
    project_dir: &Path,
    environment: &ProjectEnvironment,
) -> Result<PathBuf, ProjectConfigError> {
    let path = project_dir.join(ENVIRONMENT_FILE_NAME);
    let contents = format!(
        "project.type={}\njava.version={}\nmaven.version={}\nmaven.wrapper={}\n",
        environment.project_type.as_str(),
        environment.java_version,
        environment.maven.version,
        environment.maven.uses_wrapper
    );
    fs::write(&path, contents).map_err(|source| ProjectConfigError::Write {
        path: path.clone(),
        source,
    })?;
    Ok(path)
}

pub(super) fn load(project_dir: &Path) -> Result<ProjectEnvironment, ProjectConfigError> {
    let path = project_dir.join(ENVIRONMENT_FILE_NAME);
    let contents = fs::read_to_string(&path).map_err(|source| ProjectConfigError::Read {
        path: path.clone(),
        source,
    })?;
    let values = parse_entries(&path, &contents)?;

    let project_type = required(&path, &values, "project.type")?;
    if project_type != "maven" {
        return Err(invalid_value(&path, "project.type", project_type));
    }

    let java_value = required(&path, &values, "java.version")?;
    let java_version = java_value
        .parse::<u32>()
        .ok()
        .filter(|version| *version >= 5)
        .ok_or_else(|| invalid_value(&path, "java.version", java_value))?;

    let maven_version = required(&path, &values, "maven.version")?;
    if !is_maven_version(maven_version) {
        return Err(invalid_value(&path, "maven.version", maven_version));
    }

    let wrapper_value = required(&path, &values, "maven.wrapper")?;
    let uses_wrapper = wrapper_value
        .parse::<bool>()
        .map_err(|_| invalid_value(&path, "maven.wrapper", wrapper_value))?;

    Ok(ProjectEnvironment {
        project_type: ProjectType::Maven,
        java_version,
        maven: MavenEnvironment {
            version: maven_version.to_owned(),
            uses_wrapper,
        },
    })
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

    #[test]
    fn saves_and_loads_an_environment() {
        let directory = tempfile::tempdir().unwrap();
        let environment = ProjectEnvironment {
            project_type: ProjectType::Maven,
            java_version: 17,
            maven: MavenEnvironment {
                version: "3.9.9".to_owned(),
                uses_wrapper: true,
            },
        };

        let path = environment.save(directory.path()).unwrap();
        let loaded = ProjectEnvironment::load(directory.path()).unwrap();

        assert_eq!(path, directory.path().join(ENVIRONMENT_FILE_NAME));
        assert_eq!(loaded, environment);
        assert_eq!(
            fs::read_to_string(path).unwrap(),
            "project.type=maven\njava.version=17\nmaven.version=3.9.9\nmaven.wrapper=true\n"
        );
    }

    #[test]
    fn rejects_incomplete_and_invalid_files() {
        let directory = tempfile::tempdir().unwrap();
        let path = directory.path().join(ENVIRONMENT_FILE_NAME);
        fs::write(
            &path,
            "project.type=maven\njava.version=abc\nmaven.version=3.9.9\nmaven.wrapper=false\n",
        )
        .unwrap();

        let error = ProjectEnvironment::load(directory.path()).unwrap_err();
        assert!(matches!(
            error,
            ProjectConfigError::InvalidValue {
                key: "java.version",
                ..
            }
        ));
    }
}
