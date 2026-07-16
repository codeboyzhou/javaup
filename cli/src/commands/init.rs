use std::env;
use std::io::{self, Write};

use javaup_core::project::{ProjectDetectionEvent, ProjectEnvironment};

use super::CommandError;
use crate::output::Output;

pub(super) fn execute<Stdout, Stderr>(
    output: &mut Output<'_, Stdout, Stderr>,
) -> Result<(), CommandError>
where
    Stdout: Write,
    Stderr: Write,
{
    let project_dir = env::current_dir()?;
    let mut reporting_error = None;
    let environment = ProjectEnvironment::detect_with_observer(&project_dir, |event| {
        if reporting_error.is_none() {
            reporting_error = report_detection_event(output, event).err();
        }
    });
    let environment = environment?;
    if let Some(error) = reporting_error {
        return Err(error.into());
    }

    output.info("Saving the detected project environment")?;
    let config_path = environment.save_preserving_maven_settings(&project_dir)?;
    let maven = environment
        .maven()
        .ok_or(CommandError::UnsupportedBuildTool {
            expected: "maven",
            actual: environment.build_tool().as_str(),
        })?;

    writeln!(
        output.stdout(),
        "created {} (JDK {} at {}, Maven {}, wrapper: {})",
        config_path.display(),
        environment.java_version(),
        environment.java_home().display(),
        maven.version(),
        maven.uses_wrapper()
    )?;
    output.stdout().flush()?;
    Ok(output.success("Project environment initialized")?)
}

fn report_detection_event<Stdout, Stderr>(
    output: &mut Output<'_, Stdout, Stderr>,
    event: ProjectDetectionEvent,
) -> io::Result<()>
where
    Stdout: Write,
    Stderr: Write,
{
    match event {
        ProjectDetectionEvent::InspectingProject { project_dir } => output.info(format_args!(
            "Inspecting Maven project: {}",
            project_dir.display()
        )),
        ProjectDetectionEvent::ReadingJavaRequirements { pom_path } => output.info(format_args!(
            "Reading Java requirements: {}",
            pom_path.display()
        )),
        ProjectDetectionEvent::SearchingForJdk { major_version } => {
            output.info(format_args!("Looking for installed JDK {major_version}"))
        }
        ProjectDetectionEvent::JdkDetected {
            major_version,
            home,
        } => output.success(format_args!(
            "Found JDK {major_version}: {}",
            home.display()
        )),
        ProjectDetectionEvent::ReadingMavenWrapper { properties_path } => {
            output.info(format_args!(
                "Reading Maven Wrapper configuration: {}",
                properties_path.display()
            ))
        }
        ProjectDetectionEvent::MavenWrapperUnavailable => {
            output.warning("Maven Wrapper not found; checking Maven from PATH")
        }
        ProjectDetectionEvent::MavenDetected {
            version,
            uses_wrapper,
        } => {
            let source = if uses_wrapper { "wrapper" } else { "PATH" };
            output.success(format_args!("Found Maven {version} ({source})"))
        }
        _ => Ok(()),
    }
}

#[cfg(test)]
mod tests {
    use std::path::PathBuf;

    use super::*;
    use crate::output::OutputOptions;

    #[test]
    fn renders_detection_events_as_actionable_statuses() {
        let mut stdout = Vec::new();
        let mut stderr = Vec::new();
        let mut output = Output::new(&mut stdout, &mut stderr, OutputOptions::default());

        report_detection_event(
            &mut output,
            ProjectDetectionEvent::SearchingForJdk { major_version: 17 },
        )
        .unwrap();
        report_detection_event(
            &mut output,
            ProjectDetectionEvent::JdkDetected {
                major_version: 17,
                home: PathBuf::from("/opt/jdk-17"),
            },
        )
        .unwrap();
        report_detection_event(&mut output, ProjectDetectionEvent::MavenWrapperUnavailable)
            .unwrap();

        assert_eq!(
            String::from_utf8(stderr).unwrap(),
            "[INFO] Looking for installed JDK 17\n[SUCCESS] Found JDK 17: /opt/jdk-17\n[WARNING] Maven Wrapper not found; checking Maven from PATH\n"
        );
        assert!(stdout.is_empty());
    }
}
