use serde_json::Value;
use std::collections::HashMap;
use std::io::{BufRead, BufReader, Read, Write};
use std::process::{Child, ChildStdin, ChildStdout, Command, Stdio};
use std::sync::{Arc, Mutex};
use std::thread;
use tauri::{AppHandle, Emitter};

/// Manages the lifecycle of the Go sidecar process and provides
/// JSON-RPC 2.0 communication over stdio with Content-Length framing.
pub struct SidecarState {
    inner: Arc<Mutex<SidecarInner>>,
}

struct SidecarInner {
    #[allow(dead_code)]
    child: Option<Child>,
    stdin: Option<ChildStdin>,
    pending: HashMap<String, std::sync::mpsc::Sender<Result<Value, String>>>,
    next_id: u64,
}

impl SidecarState {
    pub fn new(app_handle: AppHandle) -> Self {
        let inner = SidecarInner {
            child: None,
            stdin: None,
            pending: HashMap::new(),
            next_id: 1,
        };
        let state = Self {
            inner: Arc::new(Mutex::new(inner)),
        };

        // Spawn the sidecar synchronously
        let inner_clone = state.inner.clone();
        if let Err(e) = spawn_sidecar(inner_clone, app_handle) {
            eprintln!("[devterm] Failed to spawn sidecar: {}", e);
        }

        state
    }

    pub async fn call(&self, method: &str, params: Value) -> Result<Value, String> {
        let (tx, rx) = std::sync::mpsc::channel();
        let id;

        {
            let mut inner = self.inner.lock().map_err(|e| e.to_string())?;
            id = inner.next_id.to_string();
            inner.next_id += 1;
            inner.pending.insert(id.clone(), tx);

            let request = serde_json::json!({
                "jsonrpc": "2.0",
                "id": id,
                "method": method,
                "params": params,
            });
            let body = serde_json::to_string(&request).map_err(|e| e.to_string())?;
            let frame = format!("Content-Length: {}\r\n\r\n{}", body.len(), body);

            if let Some(stdin) = inner.stdin.as_mut() {
                stdin.write_all(frame.as_bytes()).map_err(|e| e.to_string())?;
                stdin.flush().map_err(|e| e.to_string())?;
            } else {
                return Err("Sidecar not running".to_string());
            }
        }

        // Wait for response (with timeout)
        rx.recv_timeout(std::time::Duration::from_secs(30))
            .map_err(|e| format!("Sidecar response timeout: {}", e))?
    }

    pub async fn cancel(&self, id: &str) -> Result<(), String> {
        let mut inner = self.inner.lock().map_err(|e| e.to_string())?;
        if let Some(tx) = inner.pending.remove(id) {
            let _ = tx.send(Err("Cancelled".to_string()));
        }
        Ok(())
    }
}

fn spawn_sidecar(
    inner: Arc<Mutex<SidecarInner>>,
    app_handle: AppHandle,
) -> Result<(), String> {
    // Resolve the sidecar binary path
    let exe_dir = std::env::current_exe()
        .map_err(|e| e.to_string())?
        .parent()
        .ok_or("No parent dir")?
        .to_path_buf();

    let binary_name = "devterm-core-x86_64-pc-windows-gnu.exe";

    // In dev mode, the binary is at src-tauri/binaries/
    // In production, it's next to the exe
    let sidecar_path = if exe_dir.join(binary_name).exists() {
        exe_dir.join(binary_name)
    } else {
        // Try the binaries folder relative to src-tauri
        let dev_path = std::path::PathBuf::from(env!("CARGO_MANIFEST_DIR"))
            .join("binaries")
            .join(binary_name);
        if dev_path.exists() {
            dev_path
        } else {
            return Err(format!("Sidecar binary not found. Checked:\n  {}\n  {}",
                exe_dir.join(binary_name).display(),
                dev_path.display()
            ));
        }
    };

    eprintln!("[devterm] Spawning sidecar: {:?}", sidecar_path);

    let mut child = Command::new(&sidecar_path)
        .stdin(Stdio::piped())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .spawn()
        .map_err(|e| format!("Failed to spawn sidecar at {:?}: {}", sidecar_path, e))?;

    let stdin = child.stdin.take().ok_or("Failed to get sidecar stdin")?;
    let stdout = child.stdout.take().ok_or("Failed to get sidecar stdout")?;

    // Log stderr in background
    if let Some(stderr) = child.stderr.take() {
        thread::spawn(move || {
            let reader = BufReader::new(stderr);
            for line in reader.lines() {
                if let Ok(line) = line {
                    eprintln!("[devterm-core] {}", line);
                }
            }
        });
    }

    {
        let mut state = inner.lock().map_err(|e| e.to_string())?;
        state.child = Some(child);
        state.stdin = Some(stdin);
    }

    // Spawn a reader thread to process stdout responses/notifications
    let inner_clone = inner.clone();
    thread::spawn(move || {
        read_stdout(stdout, inner_clone, app_handle);
    });

    Ok(())
}

fn read_stdout(
    stdout: ChildStdout,
    inner: Arc<Mutex<SidecarInner>>,
    app_handle: AppHandle,
) {
    let mut reader = BufReader::new(stdout);
    let mut header_buf = String::new();

    loop {
        header_buf.clear();
        // Read Content-Length header
        match reader.read_line(&mut header_buf) {
            Ok(0) => break, // EOF
            Ok(_) => {}
            Err(_) => break,
        }

        let content_length: usize = if let Some(len_str) =
            header_buf.strip_prefix("Content-Length: ")
        {
            len_str.trim().parse().unwrap_or(0)
        } else {
            continue;
        };

        if content_length == 0 {
            continue;
        }

        // Read the empty line separator
        header_buf.clear();
        if reader.read_line(&mut header_buf).is_err() {
            break;
        }

        // Read the body
        let mut body = vec![0u8; content_length];
        if reader.read_exact(&mut body).is_err() {
            break;
        }

        let msg: Value = match serde_json::from_slice(&body) {
            Ok(v) => v,
            Err(_) => continue,
        };

        // Check if it's a response (has "id") or a notification (no "id")
        if let Some(id) = msg.get("id").and_then(|v| v.as_str()) {
            let mut state = match inner.lock() {
                Ok(s) => s,
                Err(_) => break,
            };
            if let Some(tx) = state.pending.remove(id) {
                if let Some(error) = msg.get("error") {
                    let _ = tx.send(Err(error.to_string()));
                } else {
                    let result = msg.get("result").cloned().unwrap_or(Value::Null);
                    let _ = tx.send(Ok(result));
                }
            }
        } else {
            // It's a notification — emit as Tauri event
            let method = msg
                .get("method")
                .and_then(|v| v.as_str())
                .unwrap_or("")
                .to_string();
            let params = msg.get("params").cloned().unwrap_or(Value::Null);
            let payload = serde_json::json!({ "method": method, "params": params });
            let _ = app_handle.emit("rpc-event", payload);
        }
    }

    eprintln!("[devterm] Sidecar stdout reader exited");
}
