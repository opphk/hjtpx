#!/usr/bin/env python3
import re
import sys

files = [
    'task18_rate_limiter.sh',
    'task5_api_testing.sh',
    'task17_cicd.sh'
]

for filename in files:
    try:
        with open(filename, 'r') as f:
            content = f.read()
        
        # 修复 echo "[X.X] ...)"
        pattern = r'echo "\[([0-9]+\.[0-9]+)\] ([^"]+)\)"'
        replacement = r'echo "[\1] \2"'
        content = re.sub(pattern, replacement, content)
        
        with open(filename, 'w') as f:
            f.write(content)
        
        print(f"Fixed {filename}")
    except Exception as e:
        print(f"Error fixing {filename}: {e}")

print("Done!")
