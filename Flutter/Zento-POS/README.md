# 🚀 Zento POS: Enterprise-Grade Offline Management System

**Zento POS** is a professional, local-first Point of Sale and Business Intelligence suite architected for high-reliability environments (Retail & Hospitality). Designed to eliminate cloud dependency, it ensures 100% operational uptime through a structured, resilient offline architecture.

---

### ⚡ System Overview (10x Speed Run)
https://github.com/user-attachments/assets/639c12f3-3b09-4e89-8a79-30879a9855ca

---

### 📺 Full Technical Deep-Dive
Watch the complete architectural walkthrough, covering staff workflows, admin controls, and manager oversight:
**[Watch the Technical Demo on YouTube](https://youtu.be/OcnC8EwLwik)**

#### 📌 Video Chapters:
* **0:00** - Secure Login & Role-Based Access
* **0:04** - POS Engine: Order Logic & Cart Management
* **0:54** - Admin Suite: Inventory, Reports, & System Config
* **3:30** - Manager Dashboard: Operational Oversight

---

## 💎 Core Engineering Highlights

### 🛒 High-Performance POS Engine
* **Reactive Workflow**: High-speed checkout with customizable product grids and barcode lookup.
* **Complex Ordering**: Support for modifiers, ingredient selection, and multi-state orders (Hold, Print, Checkout).
* **Global Ready**: Full **Multilingual support** including **RTL (Arabic)** interface.

### 📦 Advanced Inventory & Costing Engine
* **Granular Tracking**: Ingredient-level stock management with **Recipe-based automatic deduction**.
* **Smart Alerts**: Automated low-stock notifications and batch history tracking.
* **Valuation**: Real-time active stock valuation and discrepancy reporting.

### 🍽️ Hospitality & Operations
* **Table Management**: Full dine-in table mapping and Kitchen Order Ticket (KOT) printing.
* **Financial Integrity**: Shift tracking with cash control (Pay-ins/Pay-outs) and integrated expense logging.

### 🔒 Security & Architecture
* **RBAC (Role-Based Access Control)**: Dynamic UI rendering for Admin, Manager, and Cashier roles.
* **License Protection**: Custom RSA-based secure license activation mechanism.
* **Data Resiliency**: Built-in SQLite backup and restoration module.

---

## 🛠️ Technical Stack
* **Framework**: Flutter (Cross-platform Desktop & POS Terminal architecture)
* **State Management**: **Riverpod** (Implemented for predictable UI updates and memory efficiency)
* **Database Layer**: **SQLite** (sqflite_common_ffi) with structured relational integrity.
* **Hardware Integration**: Custom **ESC/POS Printer Service** (USB/Network auto-discovery & Cash Drawer triggering).
* **Platforms**: Windows, macOS, Linux, and Touch POS Terminals.

---

## 📂 Project Structure
* `/lib`: Core engines (POS, Inventory, Hardware, and Admin modules).
* `/admin_tools`: RSA License Key generation and system configuration utilities.
* `/scripts`: Data seeding and performance auditing tools.

---
**Developed by Mosaab benslim**
