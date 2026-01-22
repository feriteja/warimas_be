# Warimas Backend API 🚀

![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)
![GraphQL](https://img.shields.io/badge/GraphQL-Powered-e535ab?style=flat&logo=graphql)
![License](https://img.shields.io/badge/License-MIT-blue.svg)

**Warimas Backend** is a robust, scalable e-commerce API server built with **Golang**. It leverages **GraphQL** for flexible data querying and follows **Clean Architecture** principles to ensure maintainability and testability.

Designed to handle complex e-commerce flows including product variant management, session-based checkout, and secure payment processing via Xendit.

---

## 🌟 Key Features

- **🔐 Secure Authentication**: JWT-based authentication with role-based access control (Admin/User).
- **🛍️ Dynamic Product Catalog**: Support for Categories, Subcategories, and complex Product Variants (size, color, etc.).
- **🛒 Smart Checkout System**: Session-based checkout flow that handles inventory reservation, shipping fee calculation, and pricing dynamically.
- **💳 Payment Integration**: Seamless integration with **Xendit** for payment processing, invoices, and webhook handling.
- **📦 Order Lifecycle**: Comprehensive order state machine (Pending -> Paid -> Accepted -> Shipped -> Completed) with strict transition rules.
- **📍 Address Management**: User shipping address management with default settings.
- **⚡ High Performance**: Built on Go's efficient concurrency model and optimized SQL queries.

---

## 🛠️ Tech Stack

- **Language**: [Go (Golang)](https://go.dev/)
- **API Protocol**: [GraphQL](https://graphql.org/)
- **Framework**: [gqlgen](https://github.com/99designs/gqlgen)
- **Database**: PostgreSQL
- **Logging**: [Zap](https://github.com/uber-go/zap)
- **Payment Gateway**: [Xendit](https://www.xendit.co/)
- **Testing**: Testify & SQLMock

---

## 🚀 Getting Started

### Prerequisites

- **Go** (version 1.21 or higher)
- **PostgreSQL** running locally or via Docker

### Installation

1.  **Clone the repository**

    ```bash
    git clone https://github.com/yourusername/warimas-be.git
    cd warimas-be
    ```

2.  **Setup Environment Variables**
    Copy the example environment file and configure your credentials.

    ```bash
    cp example.env .env
    ```

    _Update `.env` with your Database credentials, JWT Secret, and Xendit API keys._

3.  **Install Dependencies**

    ```bash
    go mod download
    ```

4.  **Run the Server**
    ```bash
    go run cmd/server/main.go
    ```

The server will start at `http://localhost:8080` (or your configured port).

---

## 📖 API Documentation

This project uses **GraphQL Playground** for interactive API documentation and testing.

Once the server is running, navigate to:
👉 **http://localhost:8080/**

### Example Query

```graphql
query GetMyProfile {
  myProfile {
    id
    fullName
    email
    phone
    role
  }
}
```

---

## 📂 Project Structure

The project follows a modular Clean Architecture layout:

```
warimas-be/
├── cmd/                # Application entry points
├── internal/
│   ├── graph/          # GraphQL resolvers and schema
│   ├── user/           # User domain logic
│   ├── product/        # Product & Variant logic
│   ├── order/          # Order processing & Checkout sessions
│   ├── payment/        # Payment gateway integration
│   ├── address/        # Address management
│   ├── middleware/     # HTTP middlewares (Auth, CORS, Logging)
│   └── utils/          # Helper functions
└── ...
```

---

## 📄 License

This project is licensed under the MIT License.
