use std::env;
use std::ffi::OsString;
use std::fs::{self, OpenOptions};
use std::io::{self, Write};
use std::path::Path;
use std::path::PathBuf;

pub(crate) fn directory() -> Option<PathBuf> {
    if let Some(path) = env::var_os("JAVAUP_HOME").filter(|path| !path.is_empty()) {
        let path = PathBuf::from(path);
        return path.is_absolute().then_some(path);
    }

    platform_directory()
}

#[cfg(windows)]
fn platform_directory() -> Option<PathBuf> {
    env::var_os("APPDATA")
        .filter(|path| !path.is_empty())
        .map(PathBuf::from)
        .filter(|path| path.is_absolute())
        .map(|path| path.join(crate::PRODUCT_NAME))
}

#[cfg(target_os = "macos")]
fn platform_directory() -> Option<PathBuf> {
    env::var_os("HOME")
        .filter(|path| !path.is_empty())
        .map(PathBuf::from)
        .filter(|path| path.is_absolute())
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
        .filter(|path| path.is_absolute())
        .map(|path| path.join(".config").join(crate::PRODUCT_NAME))
}

#[cfg(not(any(windows, unix)))]
fn platform_directory() -> Option<PathBuf> {
    None
}

pub(crate) fn exclusive_lock(target: &Path) -> io::Result<std::fs::File> {
    let parent = target
        .parent()
        .ok_or_else(|| io::Error::new(io::ErrorKind::InvalidInput, "storage path has no parent"))?;
    fs::create_dir_all(parent)?;
    let lock_path =
        target.with_extension(match target.extension().and_then(|value| value.to_str()) {
            Some(extension) => format!("{extension}.lock"),
            None => "lock".to_owned(),
        });
    let lock = OpenOptions::new()
        .read(true)
        .write(true)
        .create(true)
        .truncate(false)
        .open(lock_path)?;
    lock.lock()?;
    Ok(lock)
}

pub(crate) fn atomic_write(path: &Path, contents: &[u8]) -> io::Result<()> {
    let parent = path
        .parent()
        .ok_or_else(|| io::Error::new(io::ErrorKind::InvalidInput, "storage path has no parent"))?;
    fs::create_dir_all(parent)?;
    let mut temporary = tempfile::NamedTempFile::new_in(parent)?;
    temporary.write_all(contents)?;
    temporary.as_file_mut().sync_all()?;
    temporary.persist(path).map_err(|error| error.error)?;
    sync_parent(parent)
}

#[cfg(unix)]
fn sync_parent(parent: &Path) -> io::Result<()> {
    std::fs::File::open(parent)?.sync_all()
}

#[cfg(not(unix))]
fn sync_parent(_parent: &Path) -> io::Result<()> {
    Ok(())
}

pub(crate) fn encode_path(path: &Path) -> String {
    encode_hex(&path_bytes(path))
}

pub(crate) fn decode_path(value: &str) -> Option<PathBuf> {
    let bytes = decode_hex(value)?;
    path_from_bytes(bytes)
}

pub(crate) fn path_identity(path: &Path) -> Vec<u8> {
    path_bytes(path)
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

fn decode_hex(value: &str) -> Option<Vec<u8>> {
    if !value.len().is_multiple_of(2) {
        return None;
    }
    value
        .as_bytes()
        .chunks_exact(2)
        .map(|digits| {
            let high = hex_value(digits[0])?;
            let low = hex_value(digits[1])?;
            Some((high << 4) | low)
        })
        .collect()
}

fn hex_value(value: u8) -> Option<u8> {
    match value {
        b'0'..=b'9' => Some(value - b'0'),
        b'a'..=b'f' => Some(value - b'a' + 10),
        b'A'..=b'F' => Some(value - b'A' + 10),
        _ => None,
    }
}

#[cfg(unix)]
fn path_bytes(path: &Path) -> Vec<u8> {
    use std::os::unix::ffi::OsStrExt;

    path.as_os_str().as_bytes().to_vec()
}

#[cfg(unix)]
fn path_from_bytes(bytes: Vec<u8>) -> Option<PathBuf> {
    use std::os::unix::ffi::OsStringExt;

    Some(PathBuf::from(OsString::from_vec(bytes)))
}

#[cfg(windows)]
fn path_bytes(path: &Path) -> Vec<u8> {
    use std::os::windows::ffi::OsStrExt;

    path.as_os_str()
        .encode_wide()
        .flat_map(u16::to_le_bytes)
        .collect()
}

#[cfg(windows)]
fn path_from_bytes(bytes: Vec<u8>) -> Option<PathBuf> {
    use std::os::windows::ffi::OsStringExt;

    if !bytes.len().is_multiple_of(2) {
        return None;
    }
    let units = bytes
        .chunks_exact(2)
        .map(|bytes| u16::from_le_bytes([bytes[0], bytes[1]]))
        .collect::<Vec<_>>();
    Some(PathBuf::from(OsString::from_wide(&units)))
}

#[cfg(not(any(unix, windows)))]
fn path_bytes(path: &Path) -> Vec<u8> {
    path.to_string_lossy().as_bytes().to_vec()
}

#[cfg(not(any(unix, windows)))]
fn path_from_bytes(bytes: Vec<u8>) -> Option<PathBuf> {
    String::from_utf8(bytes).ok().map(PathBuf::from)
}

#[cfg(test)]
mod tests {
    use std::sync::mpsc;
    use std::time::Duration;

    use proptest::prelude::*;

    use super::*;

    proptest! {
        #![proptest_config(ProptestConfig::with_cases(256))]

        #[test]
        fn hex_encoding_round_trips_arbitrary_bytes(bytes in prop::collection::vec(any::<u8>(), 0..512)) {
            prop_assert_eq!(decode_hex(&encode_hex(&bytes)), Some(bytes));
        }

        #[cfg(unix)]
        #[test]
        fn path_encoding_round_trips_arbitrary_unix_bytes(bytes in prop::collection::vec(any::<u8>(), 0..256)) {
            use std::os::unix::ffi::OsStringExt;

            let path = PathBuf::from(OsString::from_vec(bytes));
            prop_assert_eq!(decode_path(&encode_path(&path)), Some(path));
        }

        #[cfg(windows)]
        #[test]
        fn path_encoding_round_trips_arbitrary_windows_units(units in prop::collection::vec(any::<u16>(), 0..256)) {
            use std::os::windows::ffi::OsStringExt;

            let path = PathBuf::from(OsString::from_wide(&units));
            prop_assert_eq!(decode_path(&encode_path(&path)), Some(path));
        }
    }

    #[test]
    fn round_trips_native_paths() {
        let path = Path::new("some directory").join("jdk-21");
        assert_eq!(decode_path(&encode_path(&path)), Some(path));
    }

    #[cfg(unix)]
    #[test]
    fn round_trips_non_unicode_unix_paths() {
        use std::os::unix::ffi::OsStringExt;

        let path = PathBuf::from(OsString::from_vec(vec![b'/', b't', b'm', b'p', b'/', 0xff]));
        assert_eq!(decode_path(&encode_path(&path)), Some(path));
    }

    #[cfg(windows)]
    #[test]
    fn round_trips_unpaired_windows_path_units() {
        use std::os::windows::ffi::OsStringExt;

        let path = PathBuf::from(OsString::from_wide(&[
            b'C' as u16,
            b':' as u16,
            b'\\' as u16,
            0xd800,
        ]));
        assert_eq!(decode_path(&encode_path(&path)), Some(path));
    }

    #[test]
    fn atomically_replaces_existing_contents_under_a_lock() {
        let directory = tempfile::tempdir().unwrap();
        let path = directory.path().join("state.properties");
        atomic_write(&path, b"first").unwrap();

        let _lock = exclusive_lock(&path).unwrap();
        atomic_write(&path, b"second").unwrap();

        assert_eq!(fs::read(&path).unwrap(), b"second");
        assert!(path.with_extension("properties.lock").is_file());
    }

    #[test]
    fn serializes_concurrent_writers_with_an_os_lock() {
        let directory = tempfile::tempdir().unwrap();
        let path = directory.path().join("state.properties");
        let first_lock = exclusive_lock(&path).unwrap();
        let thread_path = path.clone();
        let (sender, receiver) = mpsc::channel();
        let writer = std::thread::spawn(move || {
            let _lock = exclusive_lock(&thread_path).unwrap();
            sender.send(()).unwrap();
        });

        assert!(receiver.recv_timeout(Duration::from_millis(100)).is_err());
        drop(first_lock);
        receiver.recv_timeout(Duration::from_secs(2)).unwrap();
        writer.join().unwrap();
    }
}
