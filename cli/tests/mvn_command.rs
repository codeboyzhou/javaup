use std::fs;
use std::path::{Path, PathBuf};
use std::process::{Command, Output};

struct MavenFixture {
    _directory: tempfile::TempDir,
    working_directory: PathBuf,
    java_home: PathBuf,
    javaup_home: PathBuf,
}

impl MavenFixture {
    fn new() -> Self {
        let directory = tempfile::tempdir().unwrap();
        let project = directory.path().join("project");
        let working_directory = project.join("module");
        let java_home = directory.path().join("jdk-17");
        let javaup_home = directory.path().join("javaup-home");
        fs::create_dir_all(project.join(".mvn/wrapper")).unwrap();
        fs::create_dir_all(&working_directory).unwrap();
        fs::write(
            project.join("pom.xml"),
            "<project><properties><java.version>17</java.version></properties></project>",
        )
        .unwrap();
        fs::write(
            project.join(".mvn/wrapper/maven-wrapper.properties"),
            "distributionUrl=https://example.test/apache-maven-3.9.9-bin.zip\n",
        )
        .unwrap();
        write_fake_jdk(&java_home);
        write_fake_wrapper(&project);

        let init = Command::new(env!("CARGO_BIN_EXE_jup"))
            .arg("init")
            .current_dir(&project)
            .env("JAVAUP_JDK_17_HOME", &java_home)
            .env("JAVAUP_HOME", &javaup_home)
            .env("NO_COLOR", "1")
            .output()
            .unwrap();
        assert!(init.status.success());

        Self {
            _directory: directory,
            working_directory,
            java_home,
            javaup_home,
        }
    }

    fn run(&self, arguments: &[&str]) -> Output {
        let mut command = self.command();
        command.arg("mvn").args(arguments).output().unwrap()
    }

    fn command(&self) -> Command {
        let mut command = Command::new(env!("CARGO_BIN_EXE_jup"));
        command
            .current_dir(&self.working_directory)
            .env("JAVAUP_HOME", &self.javaup_home)
            .env("NO_COLOR", "1");
        command
    }

    fn write_settings(&self, name: &str) -> PathBuf {
        let path = self._directory.path().join(name);
        fs::write(
            &path,
            "<settings xmlns=\"http://maven.apache.org/SETTINGS/1.0.0\"></settings>\n",
        )
        .unwrap();
        path
    }
}

fn write_fake_jdk(home: &Path) {
    let bin = home.join("bin");
    fs::create_dir_all(&bin).unwrap();
    fs::write(home.join("release"), "JAVA_VERSION=\"17.0.1\"\n").unwrap();
    for executable in ["java", "javac"] {
        let executable = if cfg!(windows) {
            format!("{executable}.exe")
        } else {
            executable.to_owned()
        };
        fs::write(bin.join(executable), "").unwrap();
    }
}

#[cfg(windows)]
fn write_fake_wrapper(project: &Path) {
    fs::write(
        project.join("mvnw.cmd"),
        "@echo off\r\necho ARG1=%~1\r\necho ARG2=%~2\r\necho ARG3=%~3\r\necho ARG4=%~4\r\necho CWD=%CD%\r\necho JAVA_HOME=%JAVA_HOME%\r\nif \"%~1\"==\"fail\" exit /b 23\r\nexit /b 0\r\n",
    )
    .unwrap();
}

#[cfg(unix)]
fn write_fake_wrapper(project: &Path) {
    use std::os::unix::fs::PermissionsExt;

    let wrapper = project.join("mvnw");
    fs::write(
        &wrapper,
        "#!/bin/sh\nprintf 'ARG1=%s\\nARG2=%s\\nARG3=%s\\nARG4=%s\\nCWD=%s\\nJAVA_HOME=%s\\n' \"${1-}\" \"${2-}\" \"${3-}\" \"${4-}\" \"$PWD\" \"$JAVA_HOME\"\n[ \"${1-}\" = \"fail\" ] && exit 23\nexit 0\n",
    )
    .unwrap();
    fs::set_permissions(&wrapper, fs::Permissions::from_mode(0o755)).unwrap();
}

#[test]
fn mvn_reports_the_environment_and_forwards_arguments() {
    let fixture = MavenFixture::new();
    let result = fixture.run(&["verify", "-DskipTests"]);

    assert!(
        result.status.success(),
        "status: {:?}\nstdout: {}\nstderr: {}",
        result.status,
        String::from_utf8_lossy(&result.stdout),
        String::from_utf8_lossy(&result.stderr)
    );
    assert_eq!(
        String::from_utf8(result.stdout)
            .unwrap()
            .lines()
            .collect::<Vec<_>>(),
        vec![
            "ARG1=verify".to_owned(),
            "ARG2=-DskipTests".to_owned(),
            "ARG3=".to_owned(),
            "ARG4=".to_owned(),
            format!("CWD={}", fixture.working_directory.display()),
            format!("JAVA_HOME={}", fixture.java_home.display()),
        ]
    );
    assert_eq!(
        String::from_utf8(result.stderr).unwrap().lines().collect::<Vec<_>>(),
        vec![
            "[INFO] Starting Maven command: mvn verify -DskipTests".to_owned(),
            format!(
                "[INFO] Environment: JDK 17.0.1 at {}; Maven 3.9.9 from Maven Wrapper; directory {}",
                fixture.java_home.display(),
                fixture.working_directory.display()
            ),
            "[INFO] Maven settings: Maven default".to_owned(),
            "[SUCCESS] Completed Maven command: mvn verify -DskipTests (JDK 17.0.1; Maven 3.9.9 from Maven Wrapper; settings Maven default)".to_owned(),
        ]
    );
}

#[test]
fn mvn_does_not_add_default_goals() {
    let fixture = MavenFixture::new();
    let result = fixture.run(&[]);

    assert!(
        result.status.success(),
        "status: {:?}\nstdout: {}\nstderr: {}",
        result.status,
        String::from_utf8_lossy(&result.stdout),
        String::from_utf8_lossy(&result.stderr)
    );
    let stdout = String::from_utf8(result.stdout).unwrap();
    assert_eq!(
        stdout.lines().take(2).collect::<Vec<_>>(),
        ["ARG1=", "ARG2="]
    );
    let stderr = String::from_utf8(result.stderr).unwrap();
    assert!(stderr.contains("[INFO] Starting Maven command: mvn\n"));
    assert!(!stderr.contains("clean"));
    assert!(!stderr.contains("package"));
}

#[test]
fn mvn_reports_failures_and_preserves_the_exit_code() {
    let fixture = MavenFixture::new();
    let result = fixture.run(&["fail"]);

    assert_eq!(
        result.status.code(),
        Some(23),
        "stdout: {}\nstderr: {}",
        String::from_utf8_lossy(&result.stdout),
        String::from_utf8_lossy(&result.stderr)
    );
    let stderr = String::from_utf8(result.stderr).unwrap();
    assert!(stderr.contains(
        "[ERROR] Maven command failed with exit code 23: mvn fail (JDK 17.0.1; Maven 3.9.9 from Maven Wrapper; settings Maven default)"
    ));
    assert!(!stderr.contains("[SUCCESS] Completed Maven command"));
}

#[test]
fn manages_project_settings_and_honors_command_line_overrides() {
    let fixture = MavenFixture::new();
    let settings_path = fixture.write_settings("nexus-settings.xml");
    let settings_path = fs::canonicalize(settings_path).unwrap();

    let registered = fixture
        .command()
        .args(["settings", "add", "corp-nexus"])
        .arg(&settings_path)
        .output()
        .unwrap();
    assert!(registered.status.success());
    assert!(
        String::from_utf8(registered.stderr)
            .unwrap()
            .contains("[SUCCESS] Registered Maven settings 'corp-nexus'")
    );

    let listed = fixture
        .command()
        .args(["settings", "list"])
        .output()
        .unwrap();
    assert_eq!(
        String::from_utf8(listed.stdout).unwrap(),
        format!("corp-nexus: {}\n", settings_path.display())
    );

    let bound = fixture
        .command()
        .args(["settings", "use", "corp-nexus"])
        .output()
        .unwrap();
    assert!(bound.status.success());
    assert!(
        String::from_utf8(bound.stderr)
            .unwrap()
            .contains("[SUCCESS] Bound Maven settings 'corp-nexus'")
    );

    let project_config = fs::read_dir(fixture.javaup_home.join("projects"))
        .unwrap()
        .next()
        .unwrap()
        .unwrap()
        .path();
    assert!(
        fs::read_to_string(&project_config)
            .unwrap()
            .contains("maven.settings=corp-nexus\n")
    );

    let with_profile = fixture.run(&["verify"]);
    assert!(with_profile.status.success());
    assert_eq!(
        String::from_utf8(with_profile.stdout)
            .unwrap()
            .lines()
            .take(4)
            .collect::<Vec<_>>(),
        [
            "ARG1=--settings".to_owned(),
            format!("ARG2={}", settings_path.display()),
            "ARG3=verify".to_owned(),
            "ARG4=".to_owned(),
        ]
    );
    let profile_stderr = String::from_utf8(with_profile.stderr).unwrap();
    assert!(profile_stderr.contains(&format!(
        "[INFO] Maven settings: profile 'corp-nexus' ({})",
        settings_path.display()
    )));
    assert!(profile_stderr.contains("settings profile 'corp-nexus'"));

    let project = fixture.working_directory.parent().unwrap();
    let reinitialized = fixture
        .command()
        .arg("init")
        .current_dir(project)
        .env("JAVAUP_JDK_17_HOME", &fixture.java_home)
        .output()
        .unwrap();
    assert!(reinitialized.status.success());
    assert!(
        fs::read_to_string(&project_config)
            .unwrap()
            .contains("maven.settings=corp-nexus\n")
    );

    let override_path = fixture.write_settings("override-settings.xml");
    let overridden = fixture
        .command()
        .arg("mvn")
        .arg("--settings")
        .arg(&override_path)
        .arg("verify")
        .output()
        .unwrap();
    assert!(overridden.status.success());
    assert_eq!(
        String::from_utf8(overridden.stdout)
            .unwrap()
            .lines()
            .take(4)
            .collect::<Vec<_>>(),
        [
            "ARG1=--settings".to_owned(),
            format!("ARG2={}", override_path.display()),
            "ARG3=verify".to_owned(),
            "ARG4=".to_owned(),
        ]
    );
    let overridden_stderr = String::from_utf8(overridden.stderr).unwrap();
    assert!(
        overridden_stderr.contains(
            "[WARNING] Command-line Maven settings override project settings 'corp-nexus'"
        )
    );
    assert!(overridden_stderr.contains(&format!(
        "[INFO] Maven settings: command line ({})",
        override_path.display()
    )));

    let cleared = fixture
        .command()
        .args(["settings", "clear"])
        .output()
        .unwrap();
    assert!(cleared.status.success());
    assert!(
        !fs::read_to_string(&project_config)
            .unwrap()
            .contains("maven.settings=")
    );

    let removed = fixture
        .command()
        .args(["settings", "remove", "corp-nexus"])
        .output()
        .unwrap();
    assert!(removed.status.success());
    assert!(
        String::from_utf8(removed.stderr)
            .unwrap()
            .contains("[SUCCESS] Removed Maven settings profile 'corp-nexus'")
    );

    let empty = fixture
        .command()
        .args(["settings", "list"])
        .output()
        .unwrap();
    assert_eq!(
        String::from_utf8(empty.stdout).unwrap(),
        "No Maven settings profiles registered.\n"
    );
}
