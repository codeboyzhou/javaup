use std::fs;
use std::path::Path;
use std::process::Command;

fn write_fake_jdk(home: &Path, major_version: u32) {
    let bin = home.join("bin");
    fs::create_dir_all(&bin).unwrap();
    fs::write(
        home.join("release"),
        format!("JAVA_VERSION=\"{major_version}.0.1\"\n"),
    )
    .unwrap();
    for executable in ["java", "javac"] {
        let executable = if cfg!(windows) {
            format!("{executable}.exe")
        } else {
            executable.to_owned()
        };
        fs::write(bin.join(executable), "").unwrap();
    }
}

#[test]
fn init_reports_each_stage_and_saves_the_environment() {
    let fixture = tempfile::tempdir().unwrap();
    let project = fixture.path().join("project");
    let java_home = fixture.path().join("jdk-17");
    let javaup_home = fixture.path().join("javaup-home");
    fs::create_dir_all(project.join(".mvn/wrapper")).unwrap();
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
    write_fake_jdk(&java_home, 17);

    let result = Command::new(env!("CARGO_BIN_EXE_jup"))
        .arg("init")
        .current_dir(&project)
        .env("JAVAUP_JDK_17_HOME", &java_home)
        .env("JAVAUP_HOME", &javaup_home)
        .env("NO_COLOR", "1")
        .output()
        .unwrap();

    assert!(result.status.success());
    let stdout = String::from_utf8(result.stdout).unwrap();
    let stderr = String::from_utf8(result.stderr).unwrap();
    assert!(stdout.starts_with("created "));
    assert!(stdout.contains("JDK 17"));
    assert!(stdout.contains("Maven 3.9.9, wrapper: true"));
    assert_eq!(
        stderr.lines().collect::<Vec<_>>(),
        vec![
            format!("[INFO] Inspecting Maven project: {}", project.display()),
            format!(
                "[INFO] Reading Java requirements: {}",
                project.join("pom.xml").display()
            ),
            "[INFO] Looking for installed JDK 17".to_owned(),
            format!("[SUCCESS] Found JDK 17: {}", java_home.display()),
            format!(
                "[INFO] Reading Maven Wrapper configuration: {}",
                project
                    .join(".mvn/wrapper/maven-wrapper.properties")
                    .display()
            ),
            "[SUCCESS] Found Maven 3.9.9 (wrapper)".to_owned(),
            "[INFO] Saving the detected project environment".to_owned(),
            "[SUCCESS] Project environment initialized".to_owned(),
        ]
    );

    let saved_files = fs::read_dir(javaup_home.join("projects"))
        .unwrap()
        .collect::<Result<Vec<_>, _>>()
        .unwrap();
    assert_eq!(saved_files.len(), 1);
    let saved = fs::read_to_string(saved_files[0].path()).unwrap();
    assert!(saved.contains("java.version=17\n"));
    assert!(saved.contains("maven.version=3.9.9\n"));
    assert!(saved.contains("maven.wrapper=true\n"));
}

#[test]
fn init_reports_failures_after_the_last_started_stage() {
    let project = tempfile::tempdir().unwrap();

    let result = Command::new(env!("CARGO_BIN_EXE_jup"))
        .arg("init")
        .current_dir(project.path())
        .env("NO_COLOR", "1")
        .output()
        .unwrap();

    assert_eq!(result.status.code(), Some(1));
    assert!(result.stdout.is_empty());
    let stderr = String::from_utf8(result.stderr).unwrap();
    let lines = stderr.lines().collect::<Vec<_>>();
    assert_eq!(
        lines[0],
        format!(
            "[INFO] Inspecting Maven project: {}",
            project.path().display()
        )
    );
    assert!(lines[1].starts_with("[ERROR] "));
    assert!(lines[1].contains("is not a Maven project"));
}

#[test]
fn status_reports_the_nearest_initialized_project() {
    let fixture = tempfile::tempdir().unwrap();
    let project = fixture.path().join("project");
    let child = project.join("module").join("src");
    let java_home = fixture.path().join("jdk-21");
    let javaup_home = fixture.path().join("javaup-home");
    fs::create_dir_all(project.join(".mvn/wrapper")).unwrap();
    fs::create_dir_all(&child).unwrap();
    fs::write(
        project.join("pom.xml"),
        "<project><properties><java.version>21</java.version></properties></project>",
    )
    .unwrap();
    fs::write(
        project.join(".mvn/wrapper/maven-wrapper.properties"),
        "distributionUrl=https://example.test/apache-maven-3.9.9-bin.zip\n",
    )
    .unwrap();
    write_fake_jdk(&java_home, 21);

    let init = Command::new(env!("CARGO_BIN_EXE_jup"))
        .arg("init")
        .current_dir(&project)
        .env("JAVAUP_JDK_21_HOME", &java_home)
        .env("JAVAUP_HOME", &javaup_home)
        .env("NO_COLOR", "1")
        .output()
        .unwrap();
    assert!(init.status.success());

    let configuration_path = fs::read_dir(javaup_home.join("projects"))
        .unwrap()
        .next()
        .unwrap()
        .unwrap()
        .path();
    let result = Command::new(env!("CARGO_BIN_EXE_jup"))
        .arg("status")
        .current_dir(&child)
        .env("JAVAUP_HOME", &javaup_home)
        .env("NO_COLOR", "1")
        .output()
        .unwrap();

    assert!(result.status.success());
    assert!(result.stderr.is_empty());
    assert_eq!(
        String::from_utf8(result.stdout).unwrap(),
        format!(
            "Project: {}\nConfig: {}\nBuild system: maven\nMaven version: 3.9.9 (wrapper)\nJava version: 21.0.1\nJava home: {}\n",
            project.display(),
            configuration_path.display(),
            java_home.display()
        )
    );
}
