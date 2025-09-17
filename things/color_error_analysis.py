#!/usr/bin/env python3
import numpy as np
import matplotlib.pyplot as plt
import math

# --- 1. Original Color Table Data ---
original_hex_table = [
    0xFFFFFF, 0xFFD4BF, 0xFFE9BF, 0xFFFFBF, 0xE9FFBF, 0xD4FFBF,
    0xBFFFBF, 0xBFFFD4, 0xBFFFE9, 0xBFFFFF, 0xBFE9FF, 0xBFD4FF,
    0xBFBFFF, 0xD4BFFF, 0xE9BFFF, 0xFFBFFF, 0xFFBFE9, 0xFFBFD4,
    0xFFBFBF, 0xDADADA, 0xBF9F8F, 0xBFAF8F, 0xBFBF8F, 0xAFBF8F,
    0x9FBF8F, 0x8FBF8F, 0x8FBF9F, 0x8FBFAF, 0x8FBFBF, 0x8FAFBF,
    0x8F9FBF, 0x8F8FBF, 0x9F8FBF, 0xAF8FBF, 0xBF8FBF, 0xBF8FAF,
    0xBF8F9F, 0xBF8F8F, 0xB6B6B6, 0xBF7F5F, 0xBFAF8F, 0xBFBF5F,
    0x9FBF5F, 0x7FBF5F, 0x5FBF5F, 0x5FBF7F, 0x5FBF9F, 0x5FBFBF,
    0x5F9FBF, 0x5F7FBF, 0x5F5FBF, 0x7F5FBF, 0x9F5FBF, 0xBF5FBF,
    0xBF5F9F, 0xBF5F7F, 0xBF5F5F, 0x919191, 0xBF6A3F, 0xBF943F,
    0xBFBF3F, 0x94BF3F, 0x6ABF3F, 0x3FBF3F, 0x3FBF6A, 0x3FBF94,
    0x3FBFBF, 0x3F94BF, 0x3F6ABF, 0x3F3FBF, 0x6A3FBF, 0x943FBF,
    0xBF3FBF, 0xBF3F94, 0xBF3F6A, 0xBF3F3F, 0x6D6D6D, 0xFF5500,
    0xFFAA00, 0xFFFF00, 0xAAFF00, 0x54FF00, 0x00FF00, 0x00FF54,
    0x00FFAA, 0x00FFFF, 0x00A9FF, 0x0055FF, 0x0000FF, 0x5500FF,
    0xA900FF, 0xFE00FF, 0xFF00AA, 0xFF0055, 0xFF0000, 0x484848,
    0xBF3F00, 0xBF7F00, 0xBFBF00, 0x7FBF00, 0x3FBF00, 0x00BF00,
    0x00BF3F, 0x00BF7F, 0x00BFBF, 0x007FBF, 0x003FBF, 0x0000BF,
    0x3F00BF, 0x7F00BF, 0xBF00BF, 0xBF007F, 0xBF003F, 0xBF0000,
    0x242424, 0x7F2A00, 0x7F5500, 0x7F7F00, 0x557F00, 0x2A7F00,
    0x007F00, 0x007F2A, 0x007F55, 0x007F7F, 0x00547F, 0x002A7F,
    0x00007F, 0x2A007F, 0x54007F, 0x7F007F, 0x7F0055, 0x7F002A,
    0x7F0000,
]

def hex_to_rgb(hex_val):
    r = (hex_val >> 16) & 0xFF
    g = (hex_val >> 8) & 0xFF
    b = hex_val & 0xFF
    return r, g, b

original_colors = np.array([hex_to_rgb(hex_val) for hex_val in original_hex_table], dtype=np.uint8)

# --- 2. Simplified Formula Implementation ---

hueCodes = [
    0, # Unused
    0b110100, 0b111000, 0b111100, 0b101100, 0b011100, 0b001100,
    0b001101, 0b001110, 0b001111, 0b001011, 0b000111, 0b000011,
    0b010011, 0b100011, 0b110011, 0b110010, 0b110001, 0b110000
]

Base = np.array([192, 144, 96, 64, 0, 0, 0], dtype=np.int16)
Top = np.array([256, 192, 192, 192, 256, 192, 128], dtype=np.int16)

def getValue(v, c):
    base = Base[v]
    top = Top[v]
    baseVal = base - 1 if base > 0 else 0
    topVal = top - 1
    delta = topVal - baseVal
    val = baseVal + (delta * int(c)) // 3
    return np.uint8(val)

def get_simplified_color(index):
    v = index // 19
    h = index % 19

    if h == 0:
        gray_base = 255.0 / 7.0
        gray = int(round(gray_base * (7.0 - float(v))))
        return gray, gray, gray

    code = hueCodes[h]
    rCode = (code >> 4) & 0b11
    gCode = (code >> 2) & 0b11
    bCode = code & 0b11

    r = getValue(v, rCode)
    g = getValue(v, gCode)
    b = getValue(v, bCode)
    
    return r, g, b

simplified_colors = np.array([get_simplified_color(i) for i in range(133)], dtype=np.uint8)

# --- 3. Error Calculation ---
error_matrix = simplified_colors.astype(np.int16) - original_colors.astype(np.int16)
r_error = error_matrix[:, 0]
g_error = error_matrix[:, 1]
b_error = error_matrix[:, 2]
indices = np.arange(133)

# --- 4. Plotting ---
plt.figure(figsize=(16, 7))
plt.title("Error Analysis: Simplified Formula vs. Original Table", fontsize=16, pad=20)
plt.xlabel("Color Index", fontsize=12)
plt.ylabel("Error (Simplified - Original)", fontsize=12)
plt.plot(indices, r_error, label="Red Error", color='red', alpha=0.8)
plt.plot(indices, g_error, label="Green Error", color='green', alpha=0.8)
plt.plot(indices, b_error, label="Blue Error", color='blue', alpha=0.8)
for i in range(1, 7):
    plt.axvline(x=i * 19 - 0.5, color='gray', linestyle='--', linewidth=0.8, alpha=0.7)
plt.legend()
plt.grid(True, linestyle=':', alpha=0.6)
plt.ylim(-10, 10)
plt.tight_layout()
# plt.savefig("color_error_analysis.png", dpi=150)
# plt.show()


# --- 5. Summary Statistics ---
abs_error = np.abs(error_matrix)
mae = np.mean(abs_error)
max_error = np.max(error_matrix)
min_error = np.min(error_matrix)
total_non_zero = np.count_nonzero(error_matrix)

print("\n--- Error Analysis Summary ---")
print(f"Mean Absolute Error (MAE): {mae:.4f}")
print(f"Max Positive Error: {max_error}")
print(f"Max Negative Error: {min_error}")
print(f"Total error points (non-zero): {total_non_zero} (out of {133*3} total values)")
print(f"Percentage of correct values: {100.0 * (1 - (total_non_zero / (133.0*3.0))):.2f}%")

