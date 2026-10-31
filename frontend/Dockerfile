FROM node:24-alpine AS build
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
ARG VITE_ENABLE_DEMO_TOOLS=true
ARG VITE_CLIENT_ID=anya
ENV VITE_ENABLE_DEMO_TOOLS=$VITE_ENABLE_DEMO_TOOLS
ENV VITE_CLIENT_ID=$VITE_CLIENT_ID
RUN npm run build

FROM nginx:1.28-alpine
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 80
HEALTHCHECK --interval=5s --timeout=3s --retries=10 CMD wget -qO- http://127.0.0.1/ > /dev/null || exit 1
