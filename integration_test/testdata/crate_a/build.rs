use std::env;
use std::fs;
use std::path::Path;

fn main() {
    let out_dir = env::var("OUT_DIR").unwrap();
    let dest_path = Path::new(&out_dir).join("generated_data.bin");

    // Content controlled by GENERATED_CONTENT env var, defaults to "version_1"
    let content = env::var("GENERATED_CONTENT").unwrap_or_else(|_| "version_1".to_string());
    fs::write(&dest_path, content.as_bytes()).unwrap();

    println!("cargo::rerun-if-env-changed=GENERATED_CONTENT");
}
