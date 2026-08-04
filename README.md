# url-shortener

長い URL を短縮し、短縮コード経由でリダイレクトする Web サービス。
アクセス数の計測とバックグラウンド集計、JWT によるユーザー認証を備える。

## 技術構成

| 領域 | 採用 |
|---|---|
| 言語 | Go 1.26 |
| Web フレームワーク | Echo v4 |
| DB / ORM | PostgreSQL 16 / GORM |
| マイグレーション | goose |
| 認証 | JWT (HMAC-SHA256) + bcrypt |
| ログ | 標準 `log/slog` による構造化ログ |

handler / service / repository の3層に分け、内側の層が外側の具体型に依存しない構成にしている。
依存の注入は `main.go` で行う。

## セットアップ

### 必要なもの

- Go 1.26 以降
- Docker / Docker Compose
- [goose](https://github.com/pressly/goose)

```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
```

### 1. 環境変数

```bash
cp .env.example .env
```

`JWT_SECRET` は任意のランダム文字列に変更する。`DATABASE_URL` と `JWT_SECRET` は
未設定だと起動時にエラーで終了する。

### 2. データベースの起動

開発用 (5432) とテスト用 (5433) の 2 つが立ち上がる。

```bash
make up
```

### 3. マイグレーション

```bash
make migrate
make migrate-test
```

テスト用 DB は tmpfs 上にあるため、コンテナを再起動したら `make migrate-test` を流し直す。

## 起動

```bash
make run
```

`http://localhost:8080` で待ち受ける。動作確認:

```bash
curl http://localhost:8080/health
```

## API

| メソッド | パス | 認証 | 内容 |
|---|---|---|---|
| `GET` | `/health` | — | ヘルスチェック |
| `POST` | `/signup` | — | ユーザー登録 |
| `POST` | `/login` | — | ログイン。JWT を返す |
| `GET` | `/:code` | — | 短縮 URL のリダイレクト (302) |
| `POST` | `/shorten` | 要 | URL の短縮 |
| `GET` | `/me/urls` | 要 | 自分が作成した URL の一覧 |

認証が必要なエンドポイントには `Authorization: Bearer <token>` を付ける。

```bash
curl -X POST http://localhost:8080/shorten \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com"}'
```

## テスト

```bash
make test
```

`repository` と `internal/background` は実際の PostgreSQL に接続するため、
事前に `make up` と `make migrate-test` が必要。

| コマンド | 内容 |
|---|---|
| `make test` | 全パッケージ |
| `make test-unit` | DB 不要のものだけ |
| `make test-integration` | repository のみ |
| `make test-race` | data race 検出付き |
| `make cover` | カバレッジを HTML で表示 |

## 設定

| 環境変数 | 必須 | 既定値 | 内容 |
|---|---|---|---|
| `DATABASE_URL` | ✓ | — | 接続先 DSN |
| `JWT_SECRET` | ✓ | — | JWT の署名鍵 |
| `BASE_URL` | | `http://localhost:8080` | 短縮 URL のレスポンスに使う外部向けアドレス |
| `PORT` | | `8080` | 待ち受けポート |
| `APP_ENV` | | `development` | `development` のとき SQL ログを出力する |

`BASE_URL` と `PORT` は別物である点に注意。リバースプロキシの背後では、
待ち受けは `:8080` のままでも、利用者に返す短縮 URL は公開ドメインである必要がある。

## ビルド

```bash
make build
```

Docker イメージはマルチステージビルドで、実行時は非 root ユーザーで動く。

```bash
docker build -t url-shortener .
```
