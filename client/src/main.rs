// client.rs

use tokio_tungstenite::connect_async;
use tokio::process::Command;
use futures_util::StreamExt;
use serde::Deserialize;
use std::fs;

#[derive(Deserialize)]
struct Config {
    config: String,
}

async fn update_system(config: &str) -> Result<(), std::io::Error> {
    Command::new("sh")
        .arg("-c")
        .arg(format!("git -C /etc/nixos pull && nixos-rebuild switch --flake /etc/nixos#{}", config))
        .status()
        .await?;
    Ok(())
}

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    let (ws_stream, _) = connect_async("ws://yourdomain.com:9001").await?;
    let (_, mut read) = ws_stream.split();

    let config_str = fs::read_to_string("/root/setup.toml")?;
    let config: Config = toml::from_str(&config_str)?;

    while let Some(msg) = read.next().await {
        if let Ok(tokio_tungstenite::tungstenite::protocol::Message::Text(text)) = msg {
            if text == "update" {
                update_system(&config.config).await?;
            }
        }
    }

    Ok(())
}
