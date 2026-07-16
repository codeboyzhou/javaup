use std::env;
use std::io::Write;

use javaup_core::maven_settings::MavenSettingsProfile;
use javaup_core::project::ProjectEnvironment;

use super::CommandError;
use crate::output::Output;

pub(super) fn execute<Stdout, Stderr>(
    output: &mut Output<'_, Stdout, Stderr>,
) -> Result<(), CommandError>
where
    Stdout: Write,
    Stderr: Write,
{
    let current_dir = env::current_dir()?;
    let (project_dir, environment) = ProjectEnvironment::load_nearest(&current_dir)?;
    let configuration_path = ProjectEnvironment::configuration_path(&project_dir)?;
    let maven = environment
        .maven()
        .ok_or(CommandError::UnsupportedBuildTool {
            expected: "maven",
            actual: environment.build_tool().as_str(),
        })?;
    let maven_source = if maven.uses_wrapper() {
        "wrapper"
    } else {
        "PATH"
    };
    let java_version = environment.installed_java_version()?;

    output.field("Project", project_dir.display())?;
    output.field("Config", configuration_path.display())?;
    output.field("Build system", environment.project_type().as_str())?;
    output.field(
        "Maven version",
        format_args!("{} ({maven_source})", maven.version()),
    )?;
    match maven.settings_profile() {
        Some(name) => match MavenSettingsProfile::resolve(name) {
            Ok(profile) => {
                output.field("Maven settings", profile.name())?;
                output.field("Settings path", profile.path().display())?;
            }
            Err(error) => {
                output.field("Maven settings", format_args!("{name} (unavailable)"))?;
                output.warning(error)?;
            }
        },
        None => output.field("Maven settings", "Maven default")?,
    }
    output.field("Java version", java_version)?;
    Ok(output.field("Java home", environment.java_home().display())?)
}
