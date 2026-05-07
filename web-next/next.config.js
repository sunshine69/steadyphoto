/** @type {import('next').NextConfig} */
const nextConfig = {
  // Explicitly use webpack instead of Turbopack to have better control over HMR
  webpack: true,
  
  // Configure the dev server to use polling instead of WebSockets
  // This fixes the WebSocket connection error when accessing over network IP
  devServer: {
    headers: {
      'Access-Control-Allow-Origin': '*',
      'Access-Control-Allow-Methods': 'GET, POST, PUT, DELETE, OPTIONS',
      'Access-Control-Allow-Headers': 'X-Requested-With, content-type, authorization',
    },
    devMiddleware: {
      writeToDisk: true,
    },
  },
  
  images: {
    remotePatterns: [
      {
        protocol: 'http',
        hostname: '192.168.20.23',
        port: '8081',
        pathname: '/**',
      },
      {
        protocol: 'https',
        hostname: 'picsum.photos',
        port: '',
        pathname: '/**',
      },
    ],
  },
};

export default nextConfig;
