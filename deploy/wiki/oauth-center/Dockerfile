FROM node:20-alpine
WORKDIR /app
COPY package.json ./
RUN npm install --omit=dev
COPY index.js server.js ./
COPY src ./src
ENV PORT=8300
EXPOSE 8300
CMD ["node", "server.js"]
