#!/usr/bin/env python3
import argparse
import os
import sys
from pathlib import Path

def prompt_include(rel):
    while True:
        choice = input(f"Include '{rel}'? [Y/n/a/s/q]: ").strip().lower()
        if choice == "" or choice in ("y", "yes"):
            return "yes"
        if choice in ("n", "no"):
            return "no"
        if choice in ("a", "all"):
            return "all"
        if choice in ("s", "skip"):
            return "skip"
        if choice in ("q", "quit"):
            return "quit"
        print("Please enter y / n / a (all) / s (skip all) / q (quit).")

def dump_directory(base_dir: Path, output_file: Path):
    base_dir = base_dir.resolve()
    out_path = output_file.resolve()
    include_all = False
    skip_all = False

    with out_path.open("w", encoding="utf-8") as out:
        for root, dirs, files in os.walk(base_dir, followlinks=False):
            dirs.sort()
            files.sort()
            for fname in files:
                file_path = Path(root) / fname
                try:
                    if file_path.resolve() == out_path:
                        continue
                except Exception:
                    pass

                try:
                    rel = file_path.relative_to(base_dir)
                except Exception:
                    rel = Path(os.path.relpath(file_path, base_dir))

                if skip_all:
                    include = False
                elif include_all:
                    include = True
                else:
                    action = prompt_include(rel)
                    if action == "yes":
                        include = True
                    elif action == "no":
                        include = False
                    elif action == "all":
                        include_all = True
                        include = True
                    elif action == "skip":
                        skip_all = True
                        include = False
                    elif action == "quit":
                        print("Quitting.")
                        return

                if not include:
                    continue

                out.write(f"{fname} — {rel}\n")
                out.write("```\n")
                try:
                    with file_path.open("r", encoding="utf-8", errors="replace") as fin:
                        for chunk in iter(lambda: fin.read(4096), ""):
                            out.write(chunk)
                    out.write("\n")
                except Exception as e:
                    out.write(f"[Error reading file: {e}]\n")
                out.write("```\n\n")

def main():
    parser = argparse.ArgumentParser(description="Interactively dump files from a directory into an output file.")
    parser.add_argument("-d", "--dir", dest="dir", help="Directory to scan (if omitted you'll be prompted)")
    parser.add_argument("-o", "--output", dest="output", default="output.txt", help="Output filename (default: output.txt)")
    args = parser.parse_args()

    if args.dir:
        base = Path(args.dir)
    else:
        user_input = input("Enter directory to scan: ").strip()
        base = Path(user_input or ".")

    if not base.exists() or not base.is_dir():
        print(f"Error: '{base}' is not a directory or doesn't exist.", file=sys.stderr)
        sys.exit(1)

    output_path = Path(args.output)
    dump_directory(base, output_path)
    print(f"Wrote file listing to: {output_path.resolve()}")

if __name__ == "__main__":
    main()
