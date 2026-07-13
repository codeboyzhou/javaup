//! Shared product logic used by every frontend.
//!
//! This crate must remain independent of terminal and desktop UI concerns.

#![forbid(unsafe_code)]

pub mod java;
pub mod project;

/// Product information shared by all frontends.
pub const PRODUCT_NAME: &str = "javaup";
pub const PRODUCT_DESCRIPTION: &str = "Java environment and project lifecycle manager";
