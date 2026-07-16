use std::path::Path;

#[cfg(unix)]
use std::fs;

#[cfg(unix)]
pub(crate) fn is_usable(path: &Path) -> bool {
    use std::os::unix::fs::PermissionsExt;

    fs::metadata(path)
        .is_ok_and(|metadata| metadata.is_file() && metadata.permissions().mode() & 0o111 != 0)
}

#[cfg(not(unix))]
pub(crate) fn is_usable(path: &Path) -> bool {
    path.is_file()
}
