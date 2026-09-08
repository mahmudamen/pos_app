import os
import re
import json
import hashlib

# Configuration
PROJECT_ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
LIB_DIR = os.path.join(PROJECT_ROOT, 'lib')
LANG_DIR = os.path.join(PROJECT_ROOT, 'assets', 'lang')
EN_FILE = os.path.join(LANG_DIR, 'en.json')
AR_FILE = os.path.join(LANG_DIR, 'ar.json')

# Regex to find strings: matches 'string' or "string"
STRING_REGEX = re.compile(r"(?<!import )(?<!package:)(?<!assets/)(?<!\w)(['\"])(.*?)\1")

# Strings to ignore
IGNORE_PATTERNS = [
    r"^[a-zA-Z0-9_]+$",          # Single word identifiers (often internal keys)
    r"^assets/.*",               # Asset paths
    r"^package:.*",              # Package imports
    r"^http.*",                  # URLs
    r"^{.*}$",                   # JSON-like
    r"^\d+$",                    # Numbers
    r"^/.*",                     # Paths
    r"^Error.*",                 # Error prefixes (technical)
    r"^<.*",                     # XML tags (starts with <)
    r"^SELECT .* FROM",          # SQL
    r"^INSERT INTO",             # SQL
    r"^UPDATE .* SET",           # SQL
    r"^DELETE FROM",             # SQL
    r"^CREATE TABLE",            # SQL
    r".*\.sqlite$",              # File extensions
    r".*\.db$",
    r".*\.png$",
    r".*\.jpg$",
    r".*\.json$",
    r".*\.dart$",
    r"^\$\{?[\w\.]+\}?$",        # Single variable interpolation
    r".*\$\{.*",                 # Contains ${ (Dart interpolation)
    r"^[\w\.]+\($",              # Function calls like "Store("
    r"^[\w\.]+\)$",              # Closing function calls
    r"^[^\w\s]+$",               # Only symbols
    r"^urn:.*",                  # URNs
    r"^vm:.*",                   # VM entry points
    r"^\d+:[\w:-]+$",            # Firebase IDs
    r"^([a-zA-Z0-9-]+\.)+[a-zA-Z]{2,}$", # Domain names
    r"^xmlns.*",                 # XML namespaces
    r"^id: .*",                  # toString() output
    r"^name: .*",
    r"^barcode: .*",
    r"^price: .*",
    r"^date: .*",
    r"^total: .*",
    r"^customer: .*",
    r"^updatedAt: .*",
    r"^createdAt: .*",
    r"^status: .*",
    r"^note: .*",
    r"^credit: .*",
    r"^rowid: .*",
    r"^storeId: .*",
    r"^availableQty: .*",
    r"^tryCount: .*",
    r"^lastError: .*",
    r"^linesJson: .*",
    r"^customerName: .*",
    r"^customerId: .*",
    r"^saleMan: .*",
    r"^idUser: .*",
    r"^idTStore: .*",
    r"^nameStore: .*",
    r"^FCM Registration Token:.*",
    r"^Firebase initialization error:.*",
    r"^Content-Type$",
    r"^utf-8$",
]

def load_json(filepath):
    if not os.path.exists(filepath):
        return {}
    try:
        with open(filepath, 'r', encoding='utf-8') as f:
            return json.load(f)
    except Exception as e:
        print(f"Error loading {filepath}: {e}")
        return {}

def save_json(filepath, data):
    try:
        os.makedirs(os.path.dirname(filepath), exist_ok=True)
        with open(filepath, 'w', encoding='utf-8') as f:
            json.dump(data, f, indent=2, ensure_ascii=False)
        print(f"Saved {filepath}")
    except Exception as e:
        print(f"Error saving {filepath}: {e}")

def to_key(text):
    # Create a key from text
    # 1. Try to make a semantic key from Latin characters
    key = text.lower()
    # Replace variable interpolations with placeholders or remove them
    key = re.sub(r'\$\{?[\w\.]+\}?', '', key) 
    key = re.sub(r'[^a-z0-9]+', '_', key)
    key = key.strip('_')
    
    # 2. If key is too short or empty (e.g. Arabic text), use hash
    if not key or len(key) < 3:
        hash_object = hashlib.md5(text.encode())
        return f"str_{hash_object.hexdigest()[:8]}"
        
    return key[:50]

def should_ignore(text):
    if not text.strip():
        return True
    
    # Ignore if it contains XML/HTML tags
    if '<' in text and '>' in text:
        if re.search(r'</?\w+>', text):
            return True
            
    # Ignore if it looks like code (contains many symbols)
    # Count symbols excluding spaces and Arabic chars
    symbol_count = len(re.findall(r'[^a-zA-Z0-9\s\u0600-\u06FF]', text))
    if len(text) > 0 and symbol_count / len(text) > 0.5:
        return True

    for pattern in IGNORE_PATTERNS:
        if re.match(pattern, text, re.IGNORECASE):
            return True
            
    return False

def main():
    print(f"Scanning {LIB_DIR}...")
    
    en_data = load_json(EN_FILE)
    ar_data = load_json(AR_FILE)
    
    # Cleanup phase: Remove keys that should be ignored
    keys_to_remove = []
    for key, value in en_data.items():
        if should_ignore(value):
            keys_to_remove.append(key)
        elif key.startswith('unknown_key'): # Remove old bad keys
            keys_to_remove.append(key)
        elif key.startswith('tablequ'): # Remove specific bad keys mentioned
            keys_to_remove.append(key)
            
    for key in keys_to_remove:
        if key in en_data: del en_data[key]
        if key in ar_data: del ar_data[key]
        
    if keys_to_remove:
        print(f"Removed {len(keys_to_remove)} invalid/technical keys.")

    new_strings_count = 0
    
    for root, dirs, files in os.walk(LIB_DIR):
        for file in files:
            if not file.endswith('.dart'):
                continue
                
            filepath = os.path.join(root, file)
            try:
                with open(filepath, 'r', encoding='utf-8') as f:
                    content = f.read()
            except:
                continue
                
            matches = STRING_REGEX.findall(content)
            for quote, text in matches:
                if should_ignore(text):
                    continue
                
                # Check if this string is already in our keys (by value)
                found = False
                for k, v in en_data.items():
                    if v == text:
                        found = True
                        break
                
                if not found:
                    key = to_key(text)
                    
                    # Avoid key collision
                    original_key = key
                    counter = 1
                    while key in en_data and en_data[key] != text:
                        key = f"{original_key}_{counter}"
                        counter += 1
                    
                    if key not in en_data:
                        en_data[key] = text
                        if key not in ar_data:
                            ar_data[key] = text 
                        new_strings_count += 1
                        print(f"Found new string: '{text}' -> key: '{key}'")

    if new_strings_count > 0 or keys_to_remove:
        print(f"Adding {new_strings_count} new strings...")
        save_json(EN_FILE, en_data)
        save_json(AR_FILE, ar_data)
    else:
        print("No changes.")

if __name__ == "__main__":
    main()
