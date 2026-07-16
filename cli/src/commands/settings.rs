use std::env;
use std::io::Write;

use javaup_core::maven_settings::MavenSettingsProfile;
use javaup_core::project::ProjectEnvironment;

use super::CommandError;
use crate::cli::SettingsCommand;
use crate::output::Output;

pub(super) fn execute<Stdout, Stderr>(
    command: SettingsCommand,
    output: &mut Output<'_, Stdout, Stderr>,
) -> Result<(), CommandError>
where
    Stdout: Write,
    Stderr: Write,
{
    match command {
        SettingsCommand::Add { name, path } => {
            let profile = MavenSettingsProfile::register(name, path)?;
            Ok(output.success(format_args!(
                "Registered Maven settings '{}': {}",
                profile.name(),
                profile.path().display()
            ))?)
        }
        SettingsCommand::List => {
            let profiles = MavenSettingsProfile::list()?;
            if profiles.is_empty() {
                writeln!(output.stdout(), "No Maven settings profiles registered.")?;
            } else {
                for profile in profiles {
                    output.field(profile.name(), profile.path().display())?;
                }
            }
            Ok(output.stdout().flush()?)
        }
        SettingsCommand::Use { name } => {
            let profile = MavenSettingsProfile::resolve(name)?;
            let current_dir = env::current_dir()?;
            let (project_dir, mut environment) = ProjectEnvironment::load_nearest(current_dir)?;
            if !environment.set_maven_settings(Some(&profile)) {
                return Err(CommandError::UnsupportedBuildTool {
                    expected: "maven",
                    actual: environment.build_tool().as_str(),
                });
            }
            environment.save(&project_dir)?;
            Ok(output.success(format_args!(
                "Bound Maven settings '{}' to {}",
                profile.name(),
                project_dir.display()
            ))?)
        }
        SettingsCommand::Clear => {
            let current_dir = env::current_dir()?;
            let (project_dir, mut environment) = ProjectEnvironment::load_nearest(current_dir)?;
            let maven = environment
                .maven()
                .ok_or(CommandError::UnsupportedBuildTool {
                    expected: "maven",
                    actual: environment.build_tool().as_str(),
                })?;
            let Some(profile_name) = maven.settings_profile().map(str::to_owned) else {
                return Ok(output.warning(format_args!(
                    "No Maven settings profile is bound to {}",
                    project_dir.display()
                ))?);
            };
            environment.set_maven_settings(None);
            environment.save(&project_dir)?;
            Ok(output.success(format_args!(
                "Cleared Maven settings '{profile_name}' from {}",
                project_dir.display()
            ))?)
        }
        SettingsCommand::Remove { name } => {
            let profile = MavenSettingsProfile::remove(name)?;
            output.success(format_args!(
                "Removed Maven settings profile '{}'",
                profile.name()
            ))?;
            Ok(output.warning(format_args!(
                "Existing project bindings to '{}' were not changed",
                profile.name()
            ))?)
        }
    }
}
