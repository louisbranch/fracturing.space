#!/usr/bin/env python3
"""Small TCP forwarder for exposing localhost-bound services inside a container."""

from __future__ import annotations

import argparse
import socket
import threading


def relay(source: socket.socket, dest: socket.socket) -> None:
    try:
        while True:
            data = source.recv(65536)
            if not data:
                break
            dest.sendall(data)
    except OSError:
        pass
    finally:
        try:
            dest.shutdown(socket.SHUT_WR)
        except OSError:
            pass
        try:
            source.close()
        except OSError:
            pass


def handle(client: socket.socket, target_host: str, target_port: int) -> None:
    try:
        upstream = socket.create_connection((target_host, target_port))
    except OSError:
        client.close()
        return
    threading.Thread(target=relay, args=(client, upstream), daemon=True).start()
    threading.Thread(target=relay, args=(upstream, client), daemon=True).start()


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("listen_host")
    parser.add_argument("listen_port", type=int)
    parser.add_argument("target_host")
    parser.add_argument("target_port", type=int)
    args = parser.parse_args()

    server = socket.socket()
    server.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    server.bind((args.listen_host, args.listen_port))
    server.listen()

    while True:
        client, _ = server.accept()
        threading.Thread(
            target=handle,
            args=(client, args.target_host, args.target_port),
            daemon=True,
        ).start()


if __name__ == "__main__":
    main()
