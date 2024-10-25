// client.rs

use tokio_tungstenite::connect_async;
use tokio_tungstenite::tungstenite::protocol::Message;
use serde::{Serialize, Deserialize};
use std::error::Error;
use std::time::Duration;
use tokio::time::sleep;
use futures_util::{SinkExt, StreamExt};

#[derive(Serialize, Deserialize)]
struct ClientInfo {
    ip: String,
    connected_at: String,
}

async fn register_client() -> Result<(), Box<dyn Error>> {
    let client_info = ClientInfo {
        ip: "192.168.0.5".to_string(),  // Replace with your client's IP
        connected_at: chrono::Utc::now().to_rfc3339(),
    };

    let client_json = serde_json::to_string(&client_info)?;
    let client_response = reqwest::Client::new()
        .post("http://192.168.0.5:9002/register")  // Server registration endpoint
        .body(client_json)
        .send()
        .await?;

    println!("Response: {:?}", client_response.text().await?);
    Ok(())
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn Error>> {
    let (ws_stream, _) = connect_async("ws://192.168.0.5:9001").await?;
    println!("Connected to the WebSocket server");

    // Register the client with the server
    register_client().await?;

    let (mut write, mut read) = ws_stream.split();

    // Listen for messages from the server
    tokio::spawn(async move {
        while let Some(message) = read.next().await {
            match message {
                Ok(msg) => {
                    if let Message::Text(text) = msg {
                        println!("Received: {}", text);
                    }
                },
                Err(e) => {
                    eprintln!("Error receiving message: {}", e);
                },
            }
        }
    });

    // Example of sending a periodic message to the server (optional)
    loop {
        write.send(Message::Text("Hello from client".to_string())).await?;
        sleep(Duration::from_secs(5)).await;  // Send message every 5 seconds
    }
}
