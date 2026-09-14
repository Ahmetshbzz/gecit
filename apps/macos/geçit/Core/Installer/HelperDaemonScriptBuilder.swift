import Foundation

struct HelperDaemonScriptBuilder {
    func render() -> String {
        """
        #!/bin/bash
        set -uo pipefail
        BASE="\(AppPaths.sharedDirectory)"
        CMD_FILE="$BASE/command"
        STATUS_FILE="$BASE/status.json"
        LOG_FILE="$BASE/gecit.log"
        PID_FILE="$BASE/gecit.pid"
        BINARY="\(AppPaths.installedBinaryPath)"
        MAX_LOG_BYTES=5242880
        OWNER_USER=""

        mkdir -p "$BASE"
        touch "$CMD_FILE" "$LOG_FILE"

        # The command file selects flags for a root process and the log records every
        # visited host. Keep both private to the console user; root ignores mode bits.
        refresh_owner() {
          local console_user
          console_user=$(stat -f %Su /dev/console 2>/dev/null || true)
          case "$console_user" in
            ""|root) OWNER_USER="" ;;
            *) OWNER_USER="$console_user" ;;
          esac
        }

        secure_file() {
          chmod "$2" "$1" 2>/dev/null || true
          if [ -n "$OWNER_USER" ] && [ "$(stat -f %Su "$1" 2>/dev/null || true)" != "$OWNER_USER" ]; then
            chown "$OWNER_USER" "$1" 2>/dev/null || true
          fi
        }

        secure_base() {
          refresh_owner
          secure_file "$BASE" 700
          secure_file "$CMD_FILE" 600
          secure_file "$STATUS_FILE" 600
          secure_file "$LOG_FILE" 600
          secure_file "$LOG_FILE.1" 600
          secure_file "$PID_FILE" 600
        }

        write_status() {
          local state="$1"
          local message="$2"
          local pid="null"
          if [ -f "$PID_FILE" ]; then
            local current_pid
            current_pid=$(cat "$PID_FILE" 2>/dev/null || true)
            if [ -n "$current_pid" ] && kill -0 "$current_pid" 2>/dev/null; then
              pid="$current_pid"
            fi
          fi
          cat > "$STATUS_FILE" <<EOF
        {"state":"$state","pid":$pid,"message":"$message","updatedAt":"$(date -u +%FT%TZ)"}
        EOF
          secure_file "$STATUS_FILE" 600
        }

        is_running() {
          [ -f "$PID_FILE" ] || return 1
          local current_pid
          current_pid=$(cat "$PID_FILE" 2>/dev/null || true)
          [ -n "$current_pid" ] && kill -0 "$current_pid" 2>/dev/null
        }

        rotate_log() {
          local size
          size=$(stat -f %z "$LOG_FILE" 2>/dev/null || echo 0)
          if [ "$size" -gt "$MAX_LOG_BYTES" ]; then
            mv -f "$LOG_FILE" "$LOG_FILE.1"
            : > "$LOG_FILE"
            echo "$(date -u +%FT%TZ) log rotated at $size bytes" >> "$LOG_FILE"
            secure_file "$LOG_FILE" 600
            secure_file "$LOG_FILE.1" 600
          fi
        }

        # The engine holds the log open for append, so a running session can only be
        # capped by truncating in place; moving the file would freeze the new one.
        cap_live_log() {
          local size
          size=$(stat -f %z "$LOG_FILE" 2>/dev/null || echo 0)
          if [ "$size" -gt "$MAX_LOG_BYTES" ]; then
            : > "$LOG_FILE"
            echo "$(date -u +%FT%TZ) log truncated at $size bytes" >> "$LOG_FILE"
            secure_file "$LOG_FILE" 600
          fi
        }

        SAFE_ARGS=()

        # Only the flags the app itself sends reach the root binary; a local process
        # must not be able to point the resolver at an arbitrary upstream.
        sanitize_start_args() {
          SAFE_ARGS=()
          local token value port
          for token in "$@"; do
            case "$token" in
              --fake-ttl=*)
                value="${token#--fake-ttl=}"
                case "$value" in ""|0*|*[!0-9]*) return 1 ;; esac
                if [ "$value" -gt 255 ]; then return 1; fi
                SAFE_ARGS+=("$token")
                ;;
              --doh=*)
                value="${token#--doh=}"
                case "$value" in true|false) SAFE_ARGS+=("$token") ;; *) return 1 ;; esac
                ;;
              --doh-upstream=*)
                value="${token#--doh-upstream=}"
                case "$value" in
                  cloudflare|google|quad9|nextdns|adguard) SAFE_ARGS+=("$token") ;;
                  *) return 1 ;;
                esac
                ;;
              --interface=*)
                value="${token#--interface=}"
                case "$value" in ""|*[!a-z0-9]*) return 1 ;; esac
                if [ ${#value} -gt 16 ]; then return 1; fi
                SAFE_ARGS+=("$token")
                ;;
              --ports=*)
                value="${token#--ports=}"
                case "$value" in ""|*[!0-9,]*) return 1 ;; esac
                case "$value" in ,*|*,|*,,*) return 1 ;; esac
                local IFS=,
                for port in $value; do
                  case "$port" in 0*) return 1 ;; esac
                  if [ ${#port} -gt 5 ] || [ "$port" -gt 65535 ]; then return 1; fi
                done
                SAFE_ARGS+=("$token")
                ;;
              *) return 1 ;;
            esac
          done
          return 0
        }

        start_gecit() {
          shift
          if ! sanitize_start_args "$@"; then
            echo "$(date -u +%FT%TZ) rejected startup arguments" >> "$LOG_FILE"
            write_status "error" "Geçersiz başlatma parametresi"
            return
          fi
          if is_running; then
            write_status "running" "Gecit zaten çalışıyor"
            return
          fi
          if [ ! -x "$BINARY" ]; then
            write_status "error" "Binary bulunamadı: $BINARY"
            return
          fi
          write_status "starting" "Gecit başlatılıyor"
          rotate_log
          local args=(run -v)
          if [ ${#SAFE_ARGS[@]} -gt 0 ]; then
            args+=("${SAFE_ARGS[@]}")
          fi
          "$BINARY" "${args[@]}" >> "$LOG_FILE" 2>&1 &
          echo $! > "$PID_FILE"
          secure_file "$PID_FILE" 600
          sleep 2
          if is_running; then
            write_status "running" "Gecit çalışıyor"
          else
            rm -f "$PID_FILE"
            write_status "error" "Gecit başlatılamadı"
          fi
        }

        stop_gecit() {
          write_status "stopping" "Gecit durduruluyor"
          if is_running; then
            local current_pid
            current_pid=$(cat "$PID_FILE")
            kill "$current_pid" 2>/dev/null || true
            sleep 2
            kill -9 "$current_pid" 2>/dev/null || true
          fi
          rm -f "$PID_FILE"
          "$BINARY" cleanup >> "$LOG_FILE" 2>&1 || true
          write_status "stopped" "Gecit durdu"
        }

        handle_command() {
          read -r -a parts <<< "$1"
          local command="${parts[0]}"
          case "$command" in
            start) start_gecit "${parts[@]}" ;;
            stop) stop_gecit ;;
            cleanup) stop_gecit ;;
            status) if is_running; then write_status "running" "Gecit çalışıyor"; else write_status "stopped" "Gecit durdu"; fi ;;
            *) write_status "error" "Geçersiz komut" ;;
          esac
        }

        secure_base
        write_status "stopped" "Hazır"
        while true; do
          secure_base
          if [ -s "$CMD_FILE" ]; then
            command=$(tr -d '\\r' < "$CMD_FILE" | tr -d '\\n')
            : > "$CMD_FILE"
            handle_command "$command"
          else
            if is_running; then
              write_status "running" "Gecit çalışıyor"
              cap_live_log
            fi
          fi
          sleep 1
        done
        """
    }
}
