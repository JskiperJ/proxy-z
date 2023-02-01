echo "Go Build"

go build -ldflags="-s -w" -trimpath

echo "build electron"
npm install --save-dev @electron-forge/cli
npx electron-forge import
echo "compile to package"
npm run make