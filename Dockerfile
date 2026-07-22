FROM node:20.19.1-bookworm AS front
WORKDIR /app
RUN npm install -g pnpm@10.10.0
COPY front/package.json .
COPY front/pnpm-lock.yaml .
COPY front/pnpm-workspace.yaml .
COPY front/. .
# Dependencies are only needed while generating Nuxt's static output. Keeping
# them on a temporary mount avoids retaining a large node_modules build layer
# on small production hosts.
RUN --mount=type=tmpfs,target=/app/node_modules \
    pnpm config set store-dir /app/node_modules/.pnpm-store && \
    pnpm install --frozen-lockfile && \
    pnpm run generate

FROM golang:1.23.3-alpine AS backend
ARG VERSION
ARG COMMIT_ID
WORKDIR /app
RUN apk add --no-cache build-base tzdata
COPY --from=front /app/.output/public /app/public
COPY backend/. .
RUN --mount=type=tmpfs,target=/go/pkg/mod \
    --mount=type=tmpfs,target=/root/.cache/go-build \
    --mount=type=tmpfs,target=/tmp \
    go mod download && \
    go build -tags prod -ldflags="-s -w -X main.version=${VERSION} -X main.commitId=${COMMIT_ID}" -o /app/moments

FROM alpine
WORKDIR /app/data
RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV PORT=3000
ENV TZ=Asia/Shanghai
COPY --from=backend /app/moments /app/moments
RUN chmod +x /app/moments
EXPOSE 3000
CMD ["/app/moments"]
