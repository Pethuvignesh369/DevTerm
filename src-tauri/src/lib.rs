mod sidecar;

use tauri::Manager;

#[tauri::command]
async fn rpc_call(
    method: String,
    params: serde_json::Value,
    state: tauri::State<'_, sidecar::SidecarState>,
) -> Result<serde_json::Value, String> {
    state.call(&method, params).await
}

#[tauri::command]
async fn rpc_cancel(
    id: String,
    state: tauri::State<'_, sidecar::SidecarState>,
) -> Result<(), String> {
    state.cancel(&id).await
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .setup(|app| {
            let handle = app.handle().clone();
            let sidecar_state = sidecar::SidecarState::new(handle);
            app.manage(sidecar_state);
            Ok(())
        })
        .invoke_handler(tauri::generate_handler![rpc_call, rpc_cancel])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
