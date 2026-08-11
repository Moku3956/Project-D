# デプロイ設計

## 構成図

```mermaid
graph TD
    User[ユーザー]
    Browser[ブラウザ上のReact]

    subgraph AWS
        subgraph Frontend
            CF[CloudFront]
            S3[S3\nReact静的ファイル]
        end

        subgraph Backend
            ALB[ALB]
            ECS[ECS/Fargate\nGoバックエンド]
            EFS[EFS\n.dbファイル]
        end

        ECR[ECR\nコンテナイメージ]
    end

    GHA[GitHub Actions]

    User -->|アクセス| CF
    CF --> S3
    S3 -->|静的ファイル配信| Browser
    Browser -->|API呼び出し| ALB
    ALB --> ECS
    ECS --> EFS

    GHA -->|イメージpush| ECR
    GHA -->|静的ファイルupload| S3
    ECR -->|デプロイ| ECS
```

---

## フロントエンド

- **ホスティング**: S3 + CloudFront
- **ビルド**: `npm run build` → 静的ファイルをS3にアップロード

---

## バックエンド（Go）

- **ホスティング**: ECS/Fargate
- **コンテナレジストリ**: ECR
- **永続ストレージ**: EFS（`.db` ファイルの永続化）

---

## CI/CD

- GitHub Actions で自動デプロイ
- `main` ブランチへのマージ → 本番デプロイ
- `stage` ブランチへのマージ → ステージングデプロイ

```
push to stage → ステージング環境へデプロイ
merge to main → 本番環境へデプロイ
```

---

## CORS

ReactのオリジンをGoバックエンドで許可する設定が必要。

---

## 未定事項

- ステージング環境のインフラ構成
- 認証サービス（Cognito等）の採用有無
- データベースのバックアップ戦略
