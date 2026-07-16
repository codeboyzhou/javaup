use std::env;
use std::path::PathBuf;

pub(crate) fn directory() -> Option<PathBuf> {
    if let Some(path) = env::var_os("JAVAUP_HOME").filter(|path| !path.is_empty()) {
        return Some(PathBuf::from(path));
    }

    platform_directory()
}

#[cfg(windows)]
fn platform_directory() -> Option<PathBuf> {
    env::var_os("APPDATA")
        .filter(|path| !path.is_empty())
        .map(PathBuf::from)
        .map(|path| path.join(crate::PRODUCT_NAME))
}

#[cfg(target_os = "macos")]
fn platform_directory() -> Option<PathBuf> {
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
fn platform_directory() -> Option<PathBuf> {
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
fn platform_directory() -> Option<PathBuf> {
    None
}
