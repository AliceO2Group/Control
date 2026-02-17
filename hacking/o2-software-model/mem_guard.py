import os, time, signal, datetime

TARGET_SUBSTR = os.environ.get("TARGET_SUBSTR", "/scripts/allocate.py")
ANON_THRESHOLD_KB = int(os.environ.get("ANON_THRESHOLD_KB", "800000"))
POLL_INTERVAL_S = float(os.environ.get("POLL_INTERVAL_S", "1"))
LOG_EVERY_S = float(os.environ.get("LOG_EVERY_S", "10"))

def log(msg: str):
  ts = datetime.datetime.utcnow().isoformat(timespec="seconds") + "Z"
  print(f"{ts} mem-guard: {msg}", flush=True)

def read_cmdline(pid: int) -> str:
  try:
    with open(f"/proc/{pid}/cmdline", "rb") as f:
      raw = f.read()
    return raw.replace(b"\x00", b" ").decode("utf-8", "replace").strip()
  except FileNotFoundError:
    return ""
  except PermissionError as e:
    log(f"cannot read cmdline for pid={pid}: {e}")
    return ""

def anon_kb(pid: int) -> int:
  # smaps_rollup is per-process; "Anonymous:" is in kB when present.
  path = f"/proc/{pid}/smaps_rollup"
  try:
    with open(path, "r", encoding="utf-8", errors="replace") as f:
      for line in f:
        if line.startswith("Anonymous:"):
          parts = line.split()
          return int(parts[1])
  except FileNotFoundError:
    return 0
  except PermissionError as e:
    log(f"cannot read smaps_rollup for pid={pid}: {e}")
    return 0
  except Exception as e:
    log(f"error reading smaps_rollup for pid={pid}: {e}")
    return 0
  return 0

def kill_pid(pid: int, cmd: str, akb: int):
  log(f"action=kill pid={pid} anon_kb={akb} threshold_kb={ANON_THRESHOLD_KB} cmd={cmd!r}")
  try:
    os.kill(pid, signal.SIGTERM)
    log(f"sent SIGTERM pid={pid}")
  except ProcessLookupError:
    log(f"pid={pid} already gone before SIGTERM")
    return
  except PermissionError as e:
    log(f"permission denied sending SIGTERM pid={pid}: {e}")
    return

  time.sleep(0.5)

  try:
    os.kill(pid, signal.SIGKILL)
    log(f"sent SIGKILL pid={pid}")
  except ProcessLookupError:
    log(f"pid={pid} exited after SIGTERM")
    return
  except PermissionError as e:
    log(f"permission denied sending SIGKILL pid={pid}: {e}")
    return

log(f"starting; target_substr={TARGET_SUBSTR!r} anon_threshold_kb={ANON_THRESHOLD_KB} poll_interval_s={POLL_INTERVAL_S}")
last_status = 0.0

while True:
  now = time.time()
  if now - last_status >= LOG_EVERY_S:
    log("status=ok poll=begin")
    last_status = now

  for d in os.listdir("/proc"):
    if not d.isdigit():
      continue
    pid = int(d)
    cmd = read_cmdline(pid)
    if not cmd or TARGET_SUBSTR not in cmd:
      continue

    akb = anon_kb(pid)
    if akb > ANON_THRESHOLD_KB:
      kill_pid(pid, cmd, akb)

  time.sleep(POLL_INTERVAL_S)