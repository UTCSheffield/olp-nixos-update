// server.rs

use tokio::net::TcpListener;
use tokio_tungstenite::accept_async;
use tokio_tungstenite::tungstenite::protocol::Message;
use futures_util::stream::SplitSink;
use futures_util::{SinkExt, StreamExt};
use std::sync::{Arc, Mutex};
use std::fs;
use chrono::Utc;
use warp::Filter;
use serde::{Serialize, Deserialize};
use serde_json::json;
use users::{get_user_by_uid, get_current_uid};

#[derive(Serialize, Deserialize)]
struct ClientInfo {
    ip: String,
    connected_at: String,
}

type Clients = Arc<Mutex<Vec<SplitSink<tokio_tungstenite::WebSocketStream<tokio::net::TcpStream>, Message>>>>;

async fn store_client_info(client_info: ClientInfo) -> anyhow::Result<()> {
    let file_path = "/root/update-clients.json";
    
    let mut clients = if let Ok(data) = fs::read_to_string(file_path) {
        serde_json::from_str::<Vec<ClientInfo>>(&data).unwrap_or_default()
    } else {
        Vec::new()
    };

    clients.push(client_info);
    let json_data = serde_json::to_string_pretty(&clients)?;
    fs::write(file_path, json_data)?;

    Ok(())
}

// Endpoint to handle client registration over HTTP
async fn register_client(info: ClientInfo) -> Result<impl warp::Reply, warp::Rejection> {
    store_client_info(info).await.expect("Failed to store client info");
    Ok(warp::reply::json(&json!({"status": "client registered"})))
}

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    let user = get_user_by_uid(get_current_uid()).unwrap().name().to_str();
    if !user == "root" {
        eprintln!("Error: This command must be run as root.");
    }
    
    let listener = TcpListener::bind("0.0.0.0:9001").await?;
    let clients: Clients = Arc::new(Mutex::new(Vec::new()));

    let clients_ws = clients.clone();

    tokio::spawn(async move {
        while let Ok((stream, addr)) = listener.accept().await {
            let ws_stream = accept_async(stream).await.expect("Failed to accept WebSocket connection");
            let (write, _read) = ws_stream.split();
            clients_ws.lock().unwrap().push(write);

            let client_info = ClientInfo {
                ip: addr.to_string(),
                connected_at: Utc::now().to_rfc3339(),
            };
            store_client_info(client_info).await.expect("Failed to store client info");
        }
    });

    let register_route = warp::post()
        .and(warp::path("register"))
        .and(warp::body::json())
        .and_then(register_client);

    warp::serve(register_route).run(([0, 0, 0, 0], 9002)).await;

    let update_msg = Message::Text("update".to_string());
    for client in clients.lock().unwrap().iter_mut() {
        client.send(update_msg.clone()).await?;
    }

    println!();

    Ok(())
}
