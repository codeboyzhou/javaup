//! Named Maven settings profiles stored in javaup's user-level configuration.

use std::error::Error;
use std::fmt;
use std::fs;
use std::io;
use std::path::{Path, PathBuf};

const MAVEN_DIRECTORY_NAME: &str = "maven";
const SETTINGS_DIRECTORY_NAME: &str = "settings";
const PROFILE_FILE_EXTENSION: &str = "properties";
const SETTINGS_PATH_KEY: &str = "path";

/// A named reference to a Maven `settings.xml` file.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct MavenSettingsProfile {
    name: String,
    path: PathBuf,
}

impl MavenSettingsProfile {
    /// Registers or updates a profile in javaup's user-level configuration.
    pub fn register(
        name: impl AsRef<str>,
        settings_path: impl AsRef<Path>,
    ) -> Result<Self, MavenSettingsError> {
        register_in(&storage_directory()?, name.as_ref(), settings_path.as_ref())
    }

    /// Resolves and validates a registered profile.
    pub fn resolve(name: impl AsRef<str>) -> Result<Self, MavenSettingsError> {
        resolve_in(&storage_directory()?, name.as_ref())
    }

    /// Lists all registered profiles sorted by name.
    pub fn list() -> Result<Vec<Self>, MavenSettingsError> {
        list_in(&storage_directory()?)
    }

    /// Removes a profile registration without deleting its settings file.
    pub fn remove(name: impl AsRef<str>) -> Result<Self, MavenSettingsError> {
        remove_in(&storage_directory()?, name.as_ref())
    }

    pub fn name(&self) -> &str {
        &self.name
    }

    pub fn path(&self) -> &Path {
        &self.path
    }
}

/// Error raised while managing named Maven settings profiles.
#[derive(Debug)]
#[non_exhaustive]
pub enum MavenSettingsError {
    InvalidName { name: String },
    ProfileNotFound { name: String },
    SettingsAccess { path: PathBuf, source: io::Error },
    SettingsPathNotUnicode { path: PathBuf },
    InvalidSettings { path: PathBuf, message: String },
    RegistryRead { path: PathBuf, source: io::Error },
    RegistryWrite { path: PathBuf, source: io::Error },
    RegistryRemove { path: PathBuf, source: io::Error },
    InvalidRegistry { path: PathBuf, message: String },
    StorageDirectoryUnavailable,
}

impl fmt::Display for MavenSettingsError {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::InvalidName { name } => write!(
                formatter,
                "invalid Maven settings profile name '{name}'; use lowercase letters, digits, '.', '_' or '-' (maximum 64 characters)"
            ),
            Self::ProfileNotFound { name } => {
                write!(
                    formatter,
                    "Maven settings profile '{name}' is not registered"
                )
            }
            Self::SettingsAccess { path, source } => write!(
                formatter,
                "could not access Maven settings file {}: {source}",
                path.display()
            ),
            Self::SettingsPathNotUnicode { path } => write!(
                formatter,
                "Maven settings path {} is not valid Unicode",
                path.display()
            ),
            Self::InvalidSettings { path, message } => write!(
                formatter,
                "invalid Maven settings file {}: {message}",
                path.display()
            ),
            Self::RegistryRead { path, source } => write!(
                formatter,
                "could not read Maven settings profile {}: {source}",
                path.display()
            ),
            Self::RegistryWrite { path, source } => write!(
                formatter,
                "could not write Maven settings profile {}: {source}",
                path.display()
            ),
            Self::RegistryRemove { path, source } => write!(
                formatter,
                "could not remove Maven settings profile {}: {source}",
                path.display()
            ),
            Self::InvalidRegistry { path, message } => write!(
                formatter,
                "invalid Maven settings profile {}: {message}",
                path.display()
            ),
            Self::StorageDirectoryUnavailable => write!(
                formatter,
                "could not determine the javaup storage directory; set JAVAUP_HOME"
            ),
        }
    }
}

impl Error for MavenSettingsError {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        match self {
            Self::SettingsAccess { source, .. }
            | Self::RegistryRead { source, .. }
            | Self::RegistryWrite { source, .. }
            | Self::RegistryRemove { source, .. } => Some(source),
            _ => None,
        }
    }
}

pub(crate) fn is_valid_profile_name(name: &str) -> bool {
    !name.is_empty()
        && name.len() <= 64
        && name.chars().all(|character| {
            character.is_ascii_lowercase()
                || character.is_ascii_digit()
                || matches!(character, '.' | '_' | '-')
        })
        && name
            .chars()
            .next()
            .is_some_and(|character| character.is_ascii_lowercase() || character.is_ascii_digit())
}

fn register_in(
    storage_directory: &Path,
    name: &str,
    settings_path: &Path,
) -> Result<MavenSettingsProfile, MavenSettingsError> {
    validate_name(name)?;
    let settings_path =
        fs::canonicalize(settings_path).map_err(|source| MavenSettingsError::SettingsAccess {
            path: settings_path.to_owned(),
            source,
        })?;
    validate_settings_file(&settings_path)?;
    let settings_path_value =
        settings_path
            .to_str()
            .ok_or_else(|| MavenSettingsError::SettingsPathNotUnicode {
                path: settings_path.clone(),
            })?;
    if settings_path_value.contains(['\r', '\n']) {
        return Err(MavenSettingsError::InvalidSettings {
            path: settings_path,
            message: "path contains a line break".to_owned(),
        });
    }

    let registry_path = profile_path(storage_directory, name);
    fs::create_dir_all(
        registry_path
            .parent()
            .expect("profile path must have a parent"),
    )
    .map_err(|source| MavenSettingsError::RegistryWrite {
        path: registry_path.clone(),
        source,
    })?;
    fs::write(
        &registry_path,
        format!("{SETTINGS_PATH_KEY}={settings_path_value}\n"),
    )
    .map_err(|source| MavenSettingsError::RegistryWrite {
        path: registry_path,
        source,
    })?;

    Ok(MavenSettingsProfile {
        name: name.to_owned(),
        path: settings_path,
    })
}

fn resolve_in(
    storage_directory: &Path,
    name: &str,
) -> Result<MavenSettingsProfile, MavenSettingsError> {
    let profile = read_registration_in(storage_directory, name)?;
    validate_settings_file(&profile.path)?;
    Ok(profile)
}

fn list_in(storage_directory: &Path) -> Result<Vec<MavenSettingsProfile>, MavenSettingsError> {
    let directory = profiles_directory(storage_directory);
    let entries = match fs::read_dir(&directory) {
        Ok(entries) => entries,
        Err(error) if error.kind() == io::ErrorKind::NotFound => return Ok(Vec::new()),
        Err(source) => {
            return Err(MavenSettingsError::RegistryRead {
                path: directory,
                source,
            });
        }
    };

    let mut names = Vec::new();
    for entry in entries {
        let entry = entry.map_err(|source| MavenSettingsError::RegistryRead {
            path: directory.clone(),
            source,
        })?;
        let path = entry.path();
        if path.extension().and_then(|value| value.to_str()) != Some(PROFILE_FILE_EXTENSION) {
            continue;
        }
        let name = path
            .file_stem()
            .and_then(|value| value.to_str())
            .ok_or_else(|| MavenSettingsError::InvalidRegistry {
                path: path.clone(),
                message: "profile filename is not valid Unicode".to_owned(),
            })?;
        names.push(name.to_owned());
    }
    names.sort();
    names
        .iter()
        .map(|name| read_registration_in(storage_directory, name))
        .collect()
}

fn remove_in(
    storage_directory: &Path,
    name: &str,
) -> Result<MavenSettingsProfile, MavenSettingsError> {
    let profile = read_registration_in(storage_directory, name)?;
    let registry_path = profile_path(storage_directory, name);
    fs::remove_file(&registry_path).map_err(|source| MavenSettingsError::RegistryRemove {
        path: registry_path,
        source,
    })?;
    Ok(profile)
}

fn read_registration_in(
    storage_directory: &Path,
    name: &str,
) -> Result<MavenSettingsProfile, MavenSettingsError> {
    validate_name(name)?;
    let registry_path = profile_path(storage_directory, name);
    if !registry_path.is_file() {
        return Err(MavenSettingsError::ProfileNotFound {
            name: name.to_owned(),
        });
    }
    let contents =
        fs::read_to_string(&registry_path).map_err(|source| MavenSettingsError::RegistryRead {
            path: registry_path.clone(),
            source,
        })?;
    let settings_path = parse_registry_path(&registry_path, &contents)?;
    Ok(MavenSettingsProfile {
        name: name.to_owned(),
        path: settings_path,
    })
}

fn validate_name(name: &str) -> Result<(), MavenSettingsError> {
    is_valid_profile_name(name)
        .then_some(())
        .ok_or_else(|| MavenSettingsError::InvalidName {
            name: name.to_owned(),
        })
}

fn validate_settings_file(path: &Path) -> Result<(), MavenSettingsError> {
    if !path.is_file() {
        return Err(MavenSettingsError::InvalidSettings {
            path: path.to_owned(),
            message: "path is not a file".to_owned(),
        });
    }
    let contents =
        fs::read_to_string(path).map_err(|source| MavenSettingsError::SettingsAccess {
            path: path.to_owned(),
            source,
        })?;
    let document = roxmltree::Document::parse(&contents).map_err(|error| {
        MavenSettingsError::InvalidSettings {
            path: path.to_owned(),
            message: error.to_string(),
        }
    })?;
    if document.root_element().tag_name().name() != "settings" {
        return Err(MavenSettingsError::InvalidSettings {
            path: path.to_owned(),
            message: "document root is not <settings>".to_owned(),
        });
    }
    Ok(())
}

fn parse_registry_path(
    registry_path: &Path,
    contents: &str,
) -> Result<PathBuf, MavenSettingsError> {
    let mut path = None;
    for line in contents
        .lines()
        .map(str::trim)
        .filter(|line| !line.is_empty())
    {
        let Some((key, value)) = line.split_once('=') else {
            return Err(MavenSettingsError::InvalidRegistry {
                path: registry_path.to_owned(),
                message: "expected key=value".to_owned(),
            });
        };
        if key.trim() != SETTINGS_PATH_KEY || path.is_some() || value.trim().is_empty() {
            return Err(MavenSettingsError::InvalidRegistry {
                path: registry_path.to_owned(),
                message: format!("expected exactly one non-empty '{SETTINGS_PATH_KEY}' entry"),
            });
        }
        path = Some(PathBuf::from(value.trim()));
    }
    path.ok_or_else(|| MavenSettingsError::InvalidRegistry {
        path: registry_path.to_owned(),
        message: format!("missing '{SETTINGS_PATH_KEY}' entry"),
    })
}

fn profile_path(storage_directory: &Path, name: &str) -> PathBuf {
    profiles_directory(storage_directory).join(format!("{name}.{PROFILE_FILE_EXTENSION}"))
}

fn profiles_directory(storage_directory: &Path) -> PathBuf {
    storage_directory
        .join(MAVEN_DIRECTORY_NAME)
        .join(SETTINGS_DIRECTORY_NAME)
}

fn storage_directory() -> Result<PathBuf, MavenSettingsError> {
    crate::storage::directory().ok_or(MavenSettingsError::StorageDirectoryUnavailable)
}

#[cfg(test)]
mod tests {
    use super::*;

    fn settings_file(directory: &Path, name: &str) -> PathBuf {
        let path = directory.join(name);
        fs::write(
            &path,
            "<settings xmlns=\"http://maven.apache.org/SETTINGS/1.0.0\"></settings>\n",
        )
        .unwrap();
        path
    }

    #[test]
    fn registers_lists_resolves_and_removes_profiles() {
        let storage = tempfile::tempdir().unwrap();
        let files = tempfile::tempdir().unwrap();
        let nexus = settings_file(files.path(), "nexus.xml");
        let google = settings_file(files.path(), "google.xml");

        register_in(storage.path(), "corp-nexus", &nexus).unwrap();
        register_in(storage.path(), "google-cloud", &google).unwrap();

        let profiles = list_in(storage.path()).unwrap();
        assert_eq!(
            profiles
                .iter()
                .map(|profile| profile.name())
                .collect::<Vec<_>>(),
            ["corp-nexus", "google-cloud"]
        );
        assert_eq!(
            resolve_in(storage.path(), "corp-nexus").unwrap().path(),
            fs::canonicalize(&nexus).unwrap()
        );

        let removed = remove_in(storage.path(), "corp-nexus").unwrap();
        assert_eq!(removed.name(), "corp-nexus");
        assert!(matches!(
            resolve_in(storage.path(), "corp-nexus"),
            Err(MavenSettingsError::ProfileNotFound { .. })
        ));
    }

    #[test]
    fn rejects_invalid_names_and_settings_documents() {
        let storage = tempfile::tempdir().unwrap();
        let files = tempfile::tempdir().unwrap();
        let invalid = files.path().join("invalid.xml");
        fs::write(&invalid, "<project/>\n").unwrap();

        assert!(matches!(
            register_in(storage.path(), "Corp Nexus", &invalid),
            Err(MavenSettingsError::InvalidName { .. })
        ));
        assert!(matches!(
            register_in(storage.path(), "corp-nexus", &invalid),
            Err(MavenSettingsError::InvalidSettings { .. })
        ));
    }

    #[test]
    fn reports_registered_files_that_are_no_longer_available() {
        let storage = tempfile::tempdir().unwrap();
        let files = tempfile::tempdir().unwrap();
        let settings = settings_file(files.path(), "settings.xml");
        register_in(storage.path(), "temporary", &settings).unwrap();
        let canonical_settings = fs::canonicalize(&settings).unwrap();
        fs::remove_file(&settings).unwrap();

        assert!(matches!(
            resolve_in(storage.path(), "temporary"),
            Err(MavenSettingsError::InvalidSettings { .. })
        ));

        let profiles = list_in(storage.path()).unwrap();
        assert_eq!(profiles.len(), 1);
        assert_eq!(profiles[0].name(), "temporary");
        assert_eq!(profiles[0].path(), canonical_settings);

        remove_in(storage.path(), "temporary").unwrap();
        assert!(list_in(storage.path()).unwrap().is_empty());
    }
}
