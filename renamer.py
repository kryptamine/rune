import os

directory = "/Users/alexandersatretdinov/Work/interpreter-go/test"
old_ext = ".lox"
new_ext = ".rn"

for root, _, files in os.walk(directory):
    for filename in files:
        if filename.endswith(old_ext):
            old_path = os.path.join(root, filename)
            new_path = os.path.join(root, filename[: -len(old_ext)] + new_ext)
            os.rename(old_path, new_path)
            print(f"Renamed: {old_path} -> {new_path}")
