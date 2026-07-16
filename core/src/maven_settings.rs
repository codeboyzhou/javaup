//! Named Maven settings profiles stored in javaup's user-level configuration.

use std::error::Error;
use std::fmt;
use std::fs;
use std::io;
use std::path::{Path, PathBuf};

const MAVEN_DIRECTORY_NAME: &str = "maven";
const SETTINGS_DIRECTORY_NAME: &str = "settings";
const PROFILE_FILE_EXTENSION: &str = "properties";
const SCHEMA_VERSION_KEY: &str = "schema.version";
const CURRENT_SCHEMA_VERSION: &str = "1";
const SETTINGS_PATH_KEY: &str = "path";
const SETTINGS_PATH_HEX_KEY: &str = "path.hex";

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
                "could not determine the javaup storage directory; set JAVAUP_HOME to an absolute path"
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
    let registry_path = profile_path(storage_directory, name);
    let _lock = crate::storage::exclusive_lock(&registry_path).map_err(|source| {
        MavenSettingsError::RegistryWrite {
            path: registry_path.clone(),
            source,
        }
    })?;
    crate::storage::atomic_write(
        &registry_path,
        format!(
            "{SCHEMA_VERSION_KEY}={CURRENT_SCHEMA_VERSION}\n{SETTINGS_PATH_HEX_KEY}={}\n",
            crate::storage::encode_path(&settings_path)
        )
        .as_bytes(),
    )
    .map_err(|source| MavenSettingsError::RegistryWrite {
        path: registry_path.clone(),
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
    validate_name(name)?;
    let registry_path = profile_path(storage_directory, name);
    let _lock = crate::storage::exclusive_lock(&registry_path).map_err(|source| {
        MavenSettingsError::RegistryRemove {
            path: registry_path.clone(),
            source,
        }
    })?;
    let profile = read_registration_path(name, &registry_path)?;
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
    read_registration_path(name, &registry_path)
}

fn read_registration_path(
    name: &str,
    registry_path: &Path,
) -> Result<MavenSettingsProfile, MavenSettingsError> {
    if !registry_path.is_file() {
        return Err(MavenSettingsError::ProfileNotFound {
            name: name.to_owned(),
        });
    }
    let contents =
        fs::read_to_string(registry_path).map_err(|source| MavenSettingsError::RegistryRead {
            path: registry_path.to_owned(),
            source,
        })?;
    let settings_path = parse_registry_path(registry_path, &contents)?;
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
    let mut schema_version = None;
    let mut legacy_path = None;
    let mut encoded_path = None;
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
        let key = key.trim();
        let value = value.trim();
        if value.is_empty() {
            return Err(invalid_registry(
                registry_path,
                format!("'{key}' must not be empty"),
            ));
        }
        match key {
            SCHEMA_VERSION_KEY if schema_version.is_none() => schema_version = Some(value),
            SETTINGS_PATH_KEY if legacy_path.is_none() => legacy_path = Some(value),
            SETTINGS_PATH_HEX_KEY if encoded_path.is_none() => encoded_path = Some(value),
            _ => {
                return Err(invalid_registry(
                    registry_path,
                    format!("duplicate or unknown entry '{key}'"),
                ));
            }
        }
    }
    let path = match schema_version.unwrap_or("0") {
        "0" => legacy_path
            .map(PathBuf::from)
            .ok_or_else(|| invalid_registry(registry_path, "missing 'path' entry")),
        CURRENT_SCHEMA_VERSION => encoded_path
            .and_then(crate::storage::decode_path)
            .ok_or_else(|| invalid_registry(registry_path, "invalid or missing 'path.hex' entry")),
        version => Err(invalid_registry(
            registry_path,
            format!("unsupported schema version '{version}'"),
        )),
    }?;
    if !path.is_absolute() {
        return Err(invalid_registry(
            registry_path,
            "settings path must be absolute",
        ));
    }
    Ok(path)
}

fn invalid_registry(path: &Path, message: impl Into<String>) -> MavenSettingsError {
    MavenSettingsError::InvalidRegistry {
        path: path.to_owned(),
        message: message.into(),
    }
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

    #[test]
    fn reads_legacy_registrations_and_rewrites_them_with_the_current_schema() {
        let storage = tempfile::tempdir().unwrap();
        let files = tempfile::tempdir().unwrap();
        let settings = settings_file(files.path(), "settings.xml");
        let settings = fs::canonicalize(settings).unwrap();
        let registry_path = profile_path(storage.path(), "legacy");
        fs::create_dir_all(registry_path.parent().unwrap()).unwrap();
        fs::write(&registry_path, format!("path={}\n", settings.display())).unwrap();

        assert_eq!(
            resolve_in(storage.path(), "legacy").unwrap().path(),
            settings
        );
        register_in(storage.path(), "legacy", &settings).unwrap();

        let migrated = fs::read_to_string(registry_path).unwrap();
        assert!(migrated.starts_with("schema.version=1\npath.hex="));
        assert!(!migrated.contains(&settings.display().to_string()));
    }

    #[test]
    fn rejects_unknown_registry_schema_versions() {
        let storage = tempfile::tempdir().unwrap();
        let registry_path = profile_path(storage.path(), "future");
        fs::create_dir_all(registry_path.parent().unwrap()).unwrap();
        fs::write(&registry_path, "schema.version=99\npath.hex=00\n").unwrap();

        assert!(matches!(
            read_registration_in(storage.path(), "future"),
            Err(MavenSettingsError::InvalidRegistry { .. })
        ));
    }
}
