#!/usr/bin/env python3
import argparse
import mmap
import os
import sys
import time


def parse_args():
    p = argparse.ArgumentParser(
        description="Memory load generator: private allocation + shmem(tmpfs file) mmap read/write."
    )

    # Private memory
    p.add_argument("--private-rate-bps", type=int, default=0,
                   help="Private memory allocation+fill rate in bytes/sec. 0 disables.")
    p.add_argument("--private-bytes", type=int, default=0,
                   help="Optional cap for private bytes to allocate. 0 means no cap (grow forever).")
    p.add_argument("--private-chunk-bytes", type=int, default=4 * 1024 * 1024,
                   help="Chunk size for private allocations (bytes). Default 4MiB.")
    p.add_argument("--page-touch-bytes", type=int, default=4096,
                   help="Stride for touching/filling memory to force residency. Default 4096.")

    # Shmem mmap activity
    p.add_argument("--shmem-path", type=str, default="",
                   help="Path to a shmem/tmpfs-backed file (e.g. /dev/shm/segment.bin).")
    p.add_argument("--shmem-bytes", type=int, default=0,
                   help="Ensure shmem file size is at least this many bytes (create/truncate). 0 = don't change size.")
    p.add_argument("--shmem-read-rate-bps", type=int, default=0,
                   help="Read rate from shmem mapping in bytes/sec. Only reads if > 0.")
    p.add_argument("--shmem-write-rate-bps", type=int, default=0,
                   help="Write rate to shmem mapping in bytes/sec. Only writes if > 0.")

    # Loop control
    p.add_argument("--tick-ms", type=int, default=50,
                   help="Control loop tick in milliseconds. Default 50ms.")
    p.add_argument("--report-every-s", type=float, default=2.0,
                   help="Print status every N seconds. Default 2s.")

    return p.parse_args()


def touch_buffer(buf: bytearray, stride: int):
    """Touch/dirty memory with given stride to force page allocation."""
    n = len(buf)
    if n == 0:
        return
    v = int(time.time_ns() & 0xFF)
    for i in range(0, n, stride):
        buf[i] = (buf[i] + v + (i & 0xFF)) & 0xFF


def ensure_file_size(path: str, size: int):
    fd = os.open(path, os.O_RDWR | os.O_CREAT, 0o666)
    try:
        os.ftruncate(fd, size)
    finally:
        os.close(fd)


def main():
    args = parse_args()

    for name, v in [
        ("--private-rate-bps", args.private_rate_bps),
        ("--shmem-read-rate-bps", args.shmem_read_rate_bps),
        ("--shmem-write-rate-bps", args.shmem_write_rate_bps),
    ]:
        if v < 0:
            print(f"{name} must be >= 0", file=sys.stderr)
            return 2

    if args.private_chunk_bytes <= 0:
        print("--private-chunk-bytes must be > 0", file=sys.stderr)
        return 2
    if args.page_touch_bytes <= 0:
        print("--page-touch-bytes must be > 0", file=sys.stderr)
        return 2

    tick = args.tick_ms / 1000.0
    report_every = args.report_every_s

    # Private memory state
    private_chunks: list[bytearray] = []
    private_allocated = 0

    # Shmem state
    shmem_fd = None
    shmem_mm = None
    shmem_size = 0
    shmem_r_off = 0
    shmem_w_off = 0
    shmem_read_total = 0
    shmem_written_total = 0

    need_shmem = (args.shmem_read_rate_bps > 0) or (args.shmem_write_rate_bps > 0)
    if need_shmem:
        if not args.shmem_path:
            print("shmem activity requested but --shmem-path is empty", file=sys.stderr)
            return 2

        if args.shmem_bytes > 0:
            try:
                ensure_file_size(args.shmem_path, args.shmem_bytes)
            except Exception as e:
                print(f"Failed to create/size shmem file: {e}", file=sys.stderr)
                return 2

        # For mmap write, we need RDWR; for read-only, RDONLY is enough
        open_flags = os.O_RDONLY if args.shmem_write_rate_bps == 0 else os.O_RDWR
        try:
            shmem_fd = os.open(args.shmem_path, open_flags)
            st = os.fstat(shmem_fd)
            shmem_size = st.st_size
            if shmem_size <= 0:
                raise RuntimeError("shmem file size is 0; set --shmem-bytes to something > 0")

            access = mmap.ACCESS_READ if args.shmem_write_rate_bps == 0 else mmap.ACCESS_WRITE
            shmem_mm = mmap.mmap(shmem_fd, shmem_size, access=access)
        except Exception as e:
            if shmem_fd is not None:
                try:
                    os.close(shmem_fd)
                except Exception:
                    pass
            print(f"Failed to open/mmap shmem file: {e}", file=sys.stderr)
            return 2

    last = time.monotonic()
    last_report = last

    private_budget = 0.0
    shmem_read_budget = 0.0
    shmem_write_budget = 0.0

    # A tiny accumulator so reads can't be optimized away
    checksum = 0

    try:
        while True:
            now = time.monotonic()
            dt = now - last
            last = now

            private_budget += args.private_rate_bps * dt
            shmem_read_budget += args.shmem_read_rate_bps * dt
            shmem_write_budget += args.shmem_write_rate_bps * dt

            # --- Private allocation (allocate + touch) ---
            if args.private_rate_bps > 0:
                cap = args.private_bytes
                while private_budget >= 1:
                    if cap > 0 and private_allocated >= cap:
                        private_budget = 0
                        break

                    chunk = args.private_chunk_bytes
                    if cap > 0:
                        remaining = cap - private_allocated
                        if remaining <= 0:
                            private_budget = 0
                            break
                        if remaining < chunk:
                            chunk = remaining

                    if private_budget < chunk:
                        chunk = int(private_budget)
                        if chunk < 64 * 1024:
                            break

                    buf = bytearray(chunk)
                    touch_buffer(buf, args.page_touch_bytes)
                    private_chunks.append(buf)
                    private_allocated += chunk
                    private_budget -= chunk

            # --- Shmem mmap write: touches mapped pages, should increase RES/SHR ---
            if shmem_mm is not None and args.shmem_write_rate_bps > 0:
                while shmem_write_budget >= 1:
                    to_write = int(shmem_write_budget)
                    if to_write <= 0:
                        break

                    if shmem_w_off >= shmem_size:
                        shmem_w_off = 0

                    n = min(to_write, shmem_size - shmem_w_off)
                    if n <= 0:
                        shmem_w_off = 0
                        break

                    # Touch each page once (or more if n is large) to fault pages in.
                    stride = 4096
                    v = int(time.time_ns() & 0xFF)
                    end = shmem_w_off + n
                    for off in range(shmem_w_off, end, stride):
                        shmem_mm[off] = (shmem_mm[off] + v) & 0xFF

                    shmem_w_off = end
                    shmem_written_total += n
                    shmem_write_budget -= n

                    if shmem_w_off >= shmem_size:
                        shmem_w_off = 0

            # --- Shmem mmap read: faults pages in (RES/SHR can still rise), no dirtying ---
            if shmem_mm is not None and args.shmem_read_rate_bps > 0:
                while shmem_read_budget >= 1:
                    to_read = int(shmem_read_budget)
                    if to_read <= 0:
                        break

                    if shmem_r_off >= shmem_size:
                        shmem_r_off = 0

                    n = min(to_read, shmem_size - shmem_r_off)
                    if n <= 0:
                        shmem_r_off = 0
                        break

                    stride = 4096
                    end = shmem_r_off + n
                    for off in range(shmem_r_off, end, stride):
                        checksum ^= shmem_mm[off]

                    shmem_r_off = end
                    shmem_read_total += n
                    shmem_read_budget -= n

                    if shmem_r_off >= shmem_size:
                        shmem_r_off = 0

            # --- Reporting ---
            if now - last_report >= report_every:
                last_report = now
                parts = [f"private_allocated={private_allocated}B"]
                if shmem_mm is not None:
                    parts.append(f"shmem_size={shmem_size}B")
                    parts.append(f"shmem_written_total={shmem_written_total}B")
                    parts.append(f"shmem_read_total={shmem_read_total}B")
                    parts.append(f"checksum={checksum}")
                    parts.append(f"shmem_path={args.shmem_path}")
                print(" ".join(parts), flush=True)

            time.sleep(tick)

    except KeyboardInterrupt:
        print("\nExiting.", file=sys.stderr)
    finally:
        if shmem_mm is not None:
            try:
                shmem_mm.close()
            except Exception:
                pass
        if shmem_fd is not None:
            try:
                os.close(shmem_fd)
            except Exception:
                pass

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
