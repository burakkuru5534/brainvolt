# Firebase Auth Emulator for local development (roadmap 1.2.4, ADR-0003).
# The Auth emulator is pure Node.js, so no Java runtime is needed.
FROM node:24.22.0-alpine

ARG FIREBASE_TOOLS_VERSION=15.34.0
RUN npm install --global --no-fund --no-audit "firebase-tools@${FIREBASE_TOOLS_VERSION}" \
 && npm cache clean --force

WORKDIR /emulator
COPY firebase.json ./

# Projects whose id starts with "demo-" are offline-only: the emulator never
# contacts real Firebase services and needs no credentials.
ENV FIREBASE_PROJECT_ID=demo-brainvolt
EXPOSE 9099

CMD ["sh", "-c", "exec firebase emulators:start --only auth --project \"$FIREBASE_PROJECT_ID\""]
