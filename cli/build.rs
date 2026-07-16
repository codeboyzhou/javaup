use chrono::{DateTime, Utc};

fn main() {
    println!("cargo:rerun-if-env-changed=JAVAUP_BUILD_DATE");
    println!("cargo:rerun-if-env-changed=SOURCE_DATE_EPOCH");

    let build_date = build_date();
    let version = std::env::var("CARGO_PKG_VERSION").expect("package version is not set");
    let os = std::env::var("CARGO_CFG_TARGET_OS").expect("target OS is not set");
    let arch = std::env::var("CARGO_CFG_TARGET_ARCH").expect("target architecture is not set");

    println!("cargo:rustc-env=JAVAUP_BUILD_DATE={build_date}");
    println!("cargo:rustc-env=JAVAUP_CLI_VERSION=version v{version} {os}/{arch} ({build_date})");
}

fn build_date() -> String {
    std::env::var("JAVAUP_BUILD_DATE")
        .ok()
        .filter(|value| !value.trim().is_empty())
        .or_else(|| {
            std::env::var("SOURCE_DATE_EPOCH")
                .ok()
                .and_then(|value| value.parse::<i64>().ok())
                .and_then(|timestamp| DateTime::<Utc>::from_timestamp(timestamp, 0))
                .map(|date_time| date_time.format("%Y-%m-%d").to_string())
        })
        .unwrap_or_else(|| "unknown".to_owned())
}
