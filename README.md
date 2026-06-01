# CraftForm Backend

Go REST API for CraftForm — a Google Forms-like application with Google OAuth, form building, response collection, and analytics.

## Tech Stack

| Layer | Tool |
|---|---|
| Language | Go 1.25 |
| Framework | Echo v4 |
| Database | PostgreSQL 13+ |
| Auth | Google OAuth2 + JWT |
| Driver | pgx v5 |

---

## Prerequisites

- [Go 1.25+](https://go.dev/dl/)
- [PostgreSQL 13+](https://www.postgresql.org/download/)
- A Google Cloud project with OAuth 2.0 credentials

---

## 1. Clone & Install Dependencies

```bash
git clone <your-repo-url>
cd craftform-backend

go mod tidy
```

---

## 2. Set Up Environment Variables

Copy the example file and fill in your values:

```bash
cp .env.example .env
```

Open `.env` and edit:

```env
PORT=8080

# PostgreSQL
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=yourpassword
DB_NAME=craftform
DB_SSLMODE=disable

# JWT — generate a secret with:
# openssl rand -hex 32
JWT_SECRET=your_jwt_secret_here

# Google OAuth — from console.cloud.google.com
GOOGLE_CLIENT_ID=your_google_client_id
GOOGLE_CLIENT_SECRET=your_google_client_secret
GOOGLE_REDIRECT_URL=http://localhost:8080/auth/google/callback

# Frontend URL (for CORS)
FRONTEND_URL=http://localhost:3000

# File uploads
UPLOAD_DIR=./uploads
```

### Getting Google OAuth Credentials

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a project (or select an existing one)
3. Navigate to **APIs & Services → Credentials**
4. Click **Create Credentials → OAuth 2.0 Client ID**
5. Application type: **Web application**
6. Add Authorised redirect URI: `http://localhost:8080/auth/google/callback`
7. Copy the **Client ID** and **Client Secret** into your `.env`

---

## 3. Set Up the Database

Create the database:

```bash
createdb craftform
```

Run migrations in order:

```bash
psql craftform < migrations/000001_create_users.up.sql
psql craftform < migrations/000002_create_forms.up.sql
psql craftform < migrations/000003_create_sections_questions.up.sql
psql craftform < migrations/000004_create_responses.up.sql
psql craftform < migrations/000005_add_missing_fields.up.sql
```

---

## 4. Run the Server

```bash
go run ./cmd/api
```

The server starts at `http://localhost:8080`. You should see:

```
connected to database
⇨ http server started on [::]:8080
```

### Check it's running

```bash
curl http://localhost:8080/health
# {"status":"ok"}
```

---

## 5. Build a Binary (optional)

```bash
go build -o bin/craftform ./cmd/api
./bin/craftform
```

---

## API Routes

### Auth
| Method | Path | Auth | Description |
|---|---|---|---|
| GET | `/auth/google` | — | Redirect to Google login |
| GET | `/auth/google/callback` | — | Google OAuth callback, issues JWT |
| POST | `/auth/logout` | — | Client-side logout hint |
| GET | `/api/me` | Required | Get current user ID |

### Forms
| Method | Path | Auth | Description |
|---|---|---|---|
| GET | `/api/forms` | Required | List your forms (`?q=&status=&sort=`) |
| GET | `/api/forms/public` | — | List all active public forms (`?q=`) |
| POST | `/api/forms` | Required | Create a form |
| GET | `/api/forms/:id` | Optional | Get a form (drafts visible to owner only) |
| PUT | `/api/forms/:id` | Required | Update a form |
| DELETE | `/api/forms/:id` | Required | Delete a form |
| POST | `/api/forms/:id/duplicate` | Required | Duplicate a form |

### Responses
| Method | Path | Auth | Description |
|---|---|---|---|
| POST | `/api/forms/:id/responses` | Optional | Submit a response |
| GET | `/api/forms/:id/responses` | Required | List responses (`?q=&from=&to=`) |
| GET | `/api/forms/:id/responses/summary` | Required | Analytics summary |
| GET | `/api/forms/:id/responses/:rid` | Required | Get a single response |
| DELETE | `/api/forms/:id/responses/:rid` | Required | Delete a response |

### File Upload
| Method | Path | Auth | Description |
|---|---|---|---|
| POST | `/api/upload` | Required | Upload a file (max 5 MB), returns `{ fileUrl }` |

### Static Files
Uploaded files are served at `/uploads/<filename>`.

---

## Auth Flow (Frontend)

1. Redirect user to `GET /auth/google`
2. User approves on Google
3. Backend redirects to `FRONTEND_URL/auth/callback?token=<jwt>`
4. Frontend stores the JWT (e.g. localStorage or a cookie)
5. All protected requests include the header:
   ```
   Authorization: Bearer <jwt>
   ```

---

## Query Parameters

### `GET /api/forms`
| Param | Values | Description |
|---|---|---|
| `q` | any string | Filter by title (case-insensitive) |
| `status` | `draft` / `active` / `closed` | Filter by status |
| `sort` | `newest` / `oldest` / `most_responses` / `title` | Sort order |

### `GET /api/forms/:id/responses`
| Param | Format | Description |
|---|---|---|
| `q` | any string | Filter by respondent name |
| `from` | RFC3339 e.g. `2025-01-01T00:00:00Z` | Responses submitted after |
| `to` | RFC3339 | Responses submitted before |

---

## Rolling Back Migrations

```bash
psql craftform < migrations/000005_add_missing_fields.down.sql
psql craftform < migrations/000004_create_responses.down.sql
psql craftform < migrations/000003_create_sections_questions.down.sql
psql craftform < migrations/000002_create_forms.down.sql
psql craftform < migrations/000001_create_users.down.sql
```

---

## Project Structure

```
craftform-backend/
├── cmd/api/main.go          # Entry point
├── internal/
│   ├── config/              # Environment config
│   ├── db/                  # PostgreSQL connection
│   ├── handler/             # HTTP handlers
│   ├── middleware/           # JWT auth middleware
│   ├── model/               # Structs + error types
│   ├── repository/          # SQL queries
│   └── service/             # Business logic
├── migrations/              # SQL migration files
├── uploads/                 # Uploaded files (gitignored)
├── .env.example
└── go.mod
```
