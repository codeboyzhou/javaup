use std::fmt;

pub(crate) struct BuildInfo {
    version: &'static str,
    platform: Platform,
    build_date: &'static str,
}

impl BuildInfo {
    pub(crate) fn current() -> Self {
        Self {
            version: env!("CARGO_PKG_VERSION"),
            platform: Platform::current(),
            build_date: env!("JAVAUP_BUILD_DATE"),
        }
    }
}

impl fmt::Display for BuildInfo {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(
            formatter,
            "v{} {} ({})",
            self.version, self.platform, self.build_date
        )
    }
}

struct Platform {
    os: &'static str,
    arch: &'static str,
}

impl Platform {
    fn current() -> Self {
        Self {
            os: std::env::consts::OS,
            arch: std::env::consts::ARCH,
        }
    }
}

impl fmt::Display for Platform {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(formatter, "{}/{}", self.os, self.arch)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn uses_rust_platform_names() {
        let platform = Platform::current();

        assert_eq!(platform.os, std::env::consts::OS);
        assert_eq!(platform.arch, std::env::consts::ARCH);
    }

    #[test]
    fn formats_build_info() {
        let info = BuildInfo {
            version: "1.2.3",
            platform: Platform {
                os: "windows",
                arch: "x86_64",
            },
            build_date: "2026-07-12",
        };

        assert_eq!(info.to_string(), "v1.2.3 windows/x86_64 (2026-07-12)");
    }
}
