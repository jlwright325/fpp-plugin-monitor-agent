<?php
require_once("/opt/fpp/www/common.php");

$configPath = "/home/fpp/media/config/fpp-monitor-agent.json";
$pluginDir = "/home/fpp/media/plugins/showops-agent";
$serviceName = "fpp-monitor-agent.service";
$fallbackScript = $pluginDir . "/system/fpp-monitor-agent.sh";
$versionPaths = array(
  "/opt/fpp-monitor-agent/VERSION",
  $pluginDir . "/bin/VERSION"
);

function h($value) {
  return htmlspecialchars((string)$value, ENT_QUOTES, "UTF-8");
}

function read_config($path) {
  if (!file_exists($path)) {
    return array();
  }
  $raw = file_get_contents($path);
  if ($raw === false) {
    return array();
  }
  $data = json_decode($raw, true);
  if (!is_array($data)) {
    return array();
  }
  return $data;
}

function write_config_atomic($path, $data, &$error) {
  $dir = dirname($path);
  if (!is_dir($dir)) {
    if (!mkdir($dir, 0755, true)) {
      $error = "Failed to create config directory";
      return false;
    }
  }
  $tmp = tempnam($dir, "fppmon");
  if ($tmp === false) {
    $error = "Failed to create temp file";
    return false;
  }
  $json = json_encode($data, JSON_PRETTY_PRINT | JSON_UNESCAPED_SLASHES);
  if ($json === false) {
    $error = "Failed to encode JSON";
    @unlink($tmp);
    return false;
  }
  if (file_put_contents($tmp, $json . "\n") === false) {
    $error = "Failed to write config";
    @unlink($tmp);
    return false;
  }
  if (!rename($tmp, $path)) {
    $error = "Failed to move config into place";
    @unlink($tmp);
    return false;
  }
  return true;
}

function run_cmd($cmd, &$output, &$exitCode) {
  $output = array();
  $exitCode = 0;
  exec($cmd, $output, $exitCode);
}

function is_systemd() {
  return is_dir("/run/systemd/system") && trim(shell_exec("command -v systemctl 2>/dev/null")) !== "";
}

function service_status($serviceName) {
  if (is_systemd()) {
    run_cmd("systemctl is-active " . escapeshellarg($serviceName), $output, $code);
    if ($code === 0 && isset($output[0])) {
      return trim($output[0]);
    }
    return "inactive";
  }

  run_cmd("pgrep -f fpp-monitor-agent", $output, $code);
  return $code === 0 ? "running" : "stopped";
}

function last_log_line($serviceName) {
  if (is_systemd()) {
    run_cmd("journalctl -u " . escapeshellarg($serviceName) . " -n 1 --no-pager --output=short-iso", $output, $code);
    if ($code === 0 && isset($output[0])) {
      return trim($output[0]);
    }
    return "";
  }

  $paths = array("/var/log/syslog", "/var/log/messages");
  foreach ($paths as $path) {
    if (file_exists($path)) {
      run_cmd("tail -n 1 " . escapeshellarg($path), $output, $code);
      if ($code === 0 && isset($output[0])) {
        return trim($output[0]);
      }
    }
  }
  return "";
}

function detect_agent_version($paths) {
  foreach ($paths as $path) {
    if (file_exists($path)) {
      $raw = trim(file_get_contents($path));
      if ($raw !== "") {
        return $raw;
      }
    }
  }
  return "unknown";
}

function detect_arch() {
  $arch = php_uname("m");
  if (strpos($arch, "armv7") !== false) {
    return "armv7";
  }
  if ($arch === "aarch64" || $arch === "arm64") {
    return "arm64";
  }
  return $arch !== "" ? $arch : "unknown";
}

function service_installed($serviceName, $fallbackScript, $pluginDir) {
  $systemdPath = "/etc/systemd/system/" . $serviceName;
  $systemdLibPath = "/lib/systemd/system/" . $serviceName;
  $binSystem = "/opt/fpp-monitor-agent/fpp-monitor-agent";
  $binPlugin = $pluginDir . "/bin/fpp-monitor-agent";

  return file_exists($systemdPath) ||
    file_exists($systemdLibPath) ||
    file_exists($fallbackScript) ||
    file_exists($binSystem) ||
    file_exists($binPlugin);
}

function tail_logs($serviceName, $lines) {
  if (is_systemd()) {
    run_cmd("journalctl -u " . escapeshellarg($serviceName) . " -n " . intval($lines) . " --no-pager", $output, $code);
    if ($code === 0) {
      return implode("\n", $output);
    }
    return "Failed to read journal logs.";
  }

  $paths = array("/var/log/syslog", "/var/log/messages");
  foreach ($paths as $path) {
    if (file_exists($path)) {
      run_cmd("tail -n " . intval($lines) . " " . escapeshellarg($path), $output, $code);
      if ($code === 0) {
        return implode("\n", $output);
      }
    }
  }
  return "No log source found.";
}

function restart_agent($serviceName, $fallbackScript, &$messages, &$errors) {
  if (is_systemd()) {
    run_cmd("sudo systemctl restart " . escapeshellarg($serviceName) . " 2>&1", $output, $code);
    if ($code === 0) {
      $messages[] = "Agent restarted via systemd.";
    } else {
      $detail = trim(implode("\n", $output));
      if ($detail === "") {
        $detail = "systemctl restart exited with code " . $code . ".";
      }
      $errors[] = "Failed to restart via systemd: " . $detail;
    }
    return;
  }

  run_cmd("nohup " . escapeshellarg($fallbackScript) . " >/dev/null 2>&1 &", $output, $code);
  $messages[] = "Systemd not available; fallback runner launched.";
}

$messages = array();
$errors = array();
$logs = "";

if ($_SERVER["REQUEST_METHOD"] === "POST") {
  $action = isset($_POST["action"]) ? $_POST["action"] : "";

  if ($action === "save") {
    $enrollmentToken = trim(isset($_POST["enrollment_token"]) ? $_POST["enrollment_token"] : "");
    if (empty($errors)) {
      $current = read_config($configPath);
      $updated = $current;
      if (isset($updated["api_base_url"])) {
        unset($updated["api_base_url"]);
      }
      $updated["enrollment_token"] = $enrollmentToken;

      $error = "";
      if (write_config_atomic($configPath, $updated, $error)) {
        $messages[] = "Configuration updated.";
        if ($enrollmentToken === "") {
          $messages[] = "Enrollment token is empty; device will not enroll until it is set.";
        }
        restart_agent($serviceName, $fallbackScript, $messages, $errors);
      } else {
        $errors[] = $error;
      }
    }
  } elseif ($action === "restart") {
    restart_agent($serviceName, $fallbackScript, $messages, $errors);
  } elseif ($action === "tail") {
    $logs = tail_logs($serviceName, 50);
  }
}

$config = read_config($configPath);
$status = service_status($serviceName);
$lastLog = last_log_line($serviceName);
$installed = service_installed($serviceName, $fallbackScript, $pluginDir);
$agentVersion = detect_agent_version($versionPaths);
$arch = detect_arch();
$deviceId = isset($config["device_id"]) ? $config["device_id"] : "";
$heartbeatTs = isset($config["last_heartbeat_ts"]) ? $config["last_heartbeat_ts"] : "";
$enrolled = $deviceId !== "";
$running = ($status === "active" || $status === "running");
$logs = tail_logs($serviceName, 50);

$enrollmentValue = isset($config["enrollment_token"]) ? $config["enrollment_token"] : "";
?>

<style>
:root {
  --fpp-bg: #0c1116;
  --fpp-panel: #161d24;
  --fpp-panel-2: #1f2933;
  --fpp-border: #2b3947;
  --fpp-accent: #2fd6c9;
  --fpp-accent-2: #f9a826;
  --fpp-text: #e6f0f4;
  --fpp-muted: #98a7b5;
}
.fpp-monitor-page {
  background: radial-gradient(circle at top right, rgba(47, 214, 201, 0.12), transparent 45%),
    radial-gradient(circle at bottom left, rgba(249, 168, 38, 0.12), transparent 50%),
    var(--fpp-bg);
  color: var(--fpp-text);
  padding: 18px;
  border-radius: 12px;
}
.fpp-monitor-title {
  font-family: "Avenir Next", "Montserrat", "Trebuchet MS", sans-serif;
  font-weight: 700;
  letter-spacing: 0.5px;
  margin-bottom: 12px;
}
.fpp-monitor-card {
  border: 1px solid var(--fpp-border);
  border-radius: 12px;
  padding: 16px;
  margin-bottom: 16px;
  background: linear-gradient(160deg, rgba(31, 41, 51, 0.9), rgba(22, 29, 36, 0.95));
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.2);
}
.fpp-monitor-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 14px;
}
.fpp-monitor-actions {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}
.fpp-monitor-label {
  font-weight: 600;
  margin-bottom: 4px;
  display: block;
  color: var(--fpp-muted);
  text-transform: uppercase;
  letter-spacing: 0.08em;
  font-size: 0.72rem;
}
.fpp-monitor-value {
  font-size: 1rem;
  font-weight: 600;
}
.fpp-monitor-input {
  width: 100%;
  padding: 10px 12px;
  border-radius: 8px;
  border: 1px solid var(--fpp-border);
  background: var(--fpp-panel-2);
  color: var(--fpp-text);
}
.fpp-monitor-input:focus {
  outline: 2px solid rgba(47, 214, 201, 0.45);
}
.fpp-monitor-pre {
  max-height: 380px;
  overflow: auto;
  background: #0b0f14;
  color: #d9e4ea;
  padding: 12px;
  border-radius: 10px;
  border: 1px solid var(--fpp-border);
  font-size: 0.85rem;
}
.fpp-monitor-meta {
  color: var(--fpp-muted);
  margin-top: 6px;
}
.fpp-monitor-actions .btn-primary {
  background: var(--fpp-accent);
  border: 1px solid var(--fpp-accent);
  color: #0a1117;
  font-weight: 700;
}
.fpp-monitor-actions .btn-secondary {
  background: transparent;
  border: 1px solid var(--fpp-border);
  color: var(--fpp-text);
}
</style>

<div class="container-fluid fpp-monitor-page">
  <h2 class="fpp-monitor-title">ShowOps Configuration</h2>

  <?php foreach ($messages as $msg): ?>
    <div class="alert alert-success"><?php echo h($msg); ?></div>
  <?php endforeach; ?>
  <?php foreach ($errors as $msg): ?>
    <div class="alert alert-danger"><?php echo h($msg); ?></div>
  <?php endforeach; ?>

  <div class="fpp-monitor-card">
    <h3>Connection Status</h3>
    <div class="fpp-monitor-grid">
      <div>
        <div class="fpp-monitor-label">Installed</div>
        <div class="fpp-monitor-value"><?php echo h($installed ? "yes" : "no"); ?></div>
      </div>
      <div>
        <div class="fpp-monitor-label">Agent Running</div>
        <div class="fpp-monitor-value"><?php echo h($running ? "running" : "stopped"); ?></div>
      </div>
      <div>
        <div class="fpp-monitor-label">Enrollment</div>
        <div class="fpp-monitor-value"><?php echo h($enrolled ? "enrolled" : "not enrolled"); ?></div>
      </div>
      <div>
        <div class="fpp-monitor-label">Service Status</div>
        <div class="fpp-monitor-value"><?php echo h($status); ?></div>
      </div>
      <div>
        <div class="fpp-monitor-label">Agent Version</div>
        <div class="fpp-monitor-value"><?php echo h($agentVersion); ?></div>
      </div>
      <div>
        <div class="fpp-monitor-label">Architecture</div>
        <div class="fpp-monitor-value"><?php echo h($arch); ?></div>
      </div>
      <div>
        <div class="fpp-monitor-label">Last Log Line</div>
        <div class="fpp-monitor-value"><?php echo h($lastLog !== "" ? $lastLog : "N/A"); ?></div>
      </div>
      <div>
        <div class="fpp-monitor-label">Device ID</div>
        <div class="fpp-monitor-value"><?php echo h($deviceId !== "" ? $deviceId : "N/A"); ?></div>
      </div>
      <div>
        <div class="fpp-monitor-label">Last Heartbeat</div>
        <div class="fpp-monitor-value"><?php echo h($heartbeatTs !== "" ? $heartbeatTs : "N/A"); ?></div>
      </div>
    </div>
  </div>

  <div class="fpp-monitor-card">
    <h3>Enrollment</h3>
    <form method="post">
      <input type="hidden" name="action" value="save">
      <label class="fpp-monitor-label" for="enrollment_token">Enrollment Token</label>
      <input class="fpp-monitor-input" type="text" id="enrollment_token" name="enrollment_token" value="<?php echo h($enrollmentValue); ?>">
      <small>Leave blank to clear. Enrollment tokens are one-time use.</small>
      <div class="fpp-monitor-meta">The agent will exchange this token for a device token on first enrollment.</div>

      <div class="fpp-monitor-actions" style="margin-top: 12px;">
        <button class="btn btn-primary" type="submit">Save + Restart</button>
        <button class="btn btn-secondary" type="submit" name="action" value="restart">Restart Agent</button>
      </div>
    </form>
  </div>

  <div class="fpp-monitor-card">
    <h3>Logs</h3>
    <form method="post">
      <button class="btn btn-secondary" type="submit" name="action" value="tail">Refresh Logs</button>
    </form>

    <pre class="fpp-monitor-pre"><?php echo h($logs !== "" ? $logs : "No log output available."); ?></pre>
    <div class="fpp-monitor-meta">Showing latest 50 lines.</div>
  </div>
</div>

<?php
?>
