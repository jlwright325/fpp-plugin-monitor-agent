<?php
require_once("/opt/fpp/www/common.php");
include_once("/opt/fpp/www/header.php");

$configPath = "/home/fpp/media/config/fpp-monitor-agent.json";
$pluginDir = "/home/fpp/media/plugins/showops-agent";
$serviceName = "fpp-monitor-agent.service";
$fallbackScript = $pluginDir . "/system/fpp-monitor-agent.sh";

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
    run_cmd("sudo systemctl restart " . escapeshellarg($serviceName), $output, $code);
    if ($code === 0) {
      $messages[] = "Agent restarted via systemd.";
    } else {
      $errors[] = "Failed to restart via systemd.";
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
    $apiBase = trim(isset($_POST["api_base_url"]) ? $_POST["api_base_url"] : "");
    $enrollmentToken = trim(isset($_POST["enrollment_token"]) ? $_POST["enrollment_token"] : "");
    $heartbeatInterval = trim(isset($_POST["heartbeat_interval_sec"]) ? $_POST["heartbeat_interval_sec"] : "");
    $commandInterval = trim(isset($_POST["command_poll_interval_sec"]) ? $_POST["command_poll_interval_sec"] : "");

    if ($apiBase === "" || filter_var($apiBase, FILTER_VALIDATE_URL) === false) {
      $errors[] = "API Base URL must be a valid URL.";
    }

    if (empty($errors)) {
      $current = read_config($configPath);
      $updated = $current;
      $updated["api_base_url"] = $apiBase;
      $updated["enrollment_token"] = $enrollmentToken;

      if ($heartbeatInterval !== "" && ctype_digit($heartbeatInterval)) {
        $updated["heartbeat_interval_sec"] = intval($heartbeatInterval);
      }
      if ($commandInterval !== "" && ctype_digit($commandInterval)) {
        $updated["command_poll_interval_sec"] = intval($commandInterval);
      }

      $error = "";
      if (write_config_atomic($configPath, $updated, $error)) {
        $messages[] = "Configuration saved.";
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
    $logs = tail_logs($serviceName, 200);
  }
}

$config = read_config($configPath);
$status = service_status($serviceName);
$lastLog = last_log_line($serviceName);
$deviceId = isset($config["device_id"]) ? $config["device_id"] : "";
$heartbeatTs = isset($config["last_heartbeat_ts"]) ? $config["last_heartbeat_ts"] : "";

$apiBaseValue = isset($config["api_base_url"]) ? $config["api_base_url"] : "";
$enrollmentValue = isset($config["enrollment_token"]) ? $config["enrollment_token"] : "";
$heartbeatValue = isset($config["heartbeat_interval_sec"]) ? $config["heartbeat_interval_sec"] : "";
$commandValue = isset($config["command_poll_interval_sec"]) ? $config["command_poll_interval_sec"] : "";
?>

<style>
.fpp-monitor-card {
  border: 1px solid #d9d9d9;
  border-radius: 6px;
  padding: 16px;
  margin-bottom: 16px;
  background: #fff;
}
.fpp-monitor-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
  gap: 12px;
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
}
.fpp-monitor-input {
  width: 100%;
  padding: 8px;
}
.fpp-monitor-pre {
  max-height: 380px;
  overflow: auto;
  background: #111;
  color: #eee;
  padding: 12px;
  border-radius: 6px;
}
</style>

<div class="container-fluid">
  <h2>ShowOps Configuration</h2>

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
        <div class="fpp-monitor-label">Service Status</div>
        <div><?php echo h($status); ?></div>
      </div>
      <div>
        <div class="fpp-monitor-label">Last Log Line</div>
        <div><?php echo h($lastLog !== "" ? $lastLog : "N/A"); ?></div>
      </div>
      <div>
        <div class="fpp-monitor-label">Device ID</div>
        <div><?php echo h($deviceId !== "" ? $deviceId : "N/A"); ?></div>
      </div>
      <div>
        <div class="fpp-monitor-label">Last Heartbeat</div>
        <div><?php echo h($heartbeatTs !== "" ? $heartbeatTs : "N/A"); ?></div>
      </div>
    </div>
  </div>

  <div class="fpp-monitor-card">
    <h3>Enrollment</h3>
    <form method="post">
      <input type="hidden" name="action" value="save">
      <label class="fpp-monitor-label" for="api_base_url">API Base URL</label>
      <input class="fpp-monitor-input" type="text" id="api_base_url" name="api_base_url" value="<?php echo h($apiBaseValue); ?>" required>

      <label class="fpp-monitor-label" for="enrollment_token">Enrollment Token</label>
      <input class="fpp-monitor-input" type="text" id="enrollment_token" name="enrollment_token" value="<?php echo h($enrollmentValue); ?>">
      <small>Leave blank to clear. Enrollment tokens are one-time use.</small>

      <div class="fpp-monitor-grid" style="margin-top: 12px;">
        <div>
          <label class="fpp-monitor-label" for="heartbeat_interval_sec">Heartbeat Interval (sec)</label>
          <input class="fpp-monitor-input" type="number" min="1" id="heartbeat_interval_sec" name="heartbeat_interval_sec" value="<?php echo h($heartbeatValue); ?>">
        </div>
        <div>
          <label class="fpp-monitor-label" for="command_poll_interval_sec">Command Poll Interval (sec)</label>
          <input class="fpp-monitor-input" type="number" min="1" id="command_poll_interval_sec" name="command_poll_interval_sec" value="<?php echo h($commandValue); ?>">
        </div>
      </div>

      <div class="fpp-monitor-actions" style="margin-top: 12px;">
        <button class="btn btn-primary" type="submit">Save + Restart</button>
        <button class="btn btn-secondary" type="submit" name="action" value="restart">Restart Agent</button>
      </div>
    </form>
  </div>

  <div class="fpp-monitor-card">
    <h3>Debug</h3>
    <form method="post">
      <button class="btn btn-secondary" type="submit" name="action" value="tail">Tail Logs</button>
    </form>

    <?php if ($logs !== ""): ?>
      <pre class="fpp-monitor-pre"><?php echo h($logs); ?></pre>
    <?php endif; ?>
  </div>
</div>

<?php
include_once("/opt/fpp/www/footer.php");
?>
