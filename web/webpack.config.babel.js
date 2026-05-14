const webpack = require('webpack');
const path = require('path');

const CompressionWebpackPlugin = require('compression-webpack-plugin');

const debug = process.env.NODE_ENV !== 'production';

module.exports = {
  mode: debug ? 'development' : 'production',
  entry: [
    path.join(__dirname, 'src', 'index.js'),
  ],
  output: {
    path: path.join(__dirname, 'dist'),
    filename: '[name].js',
    chunkFilename: '[name].js',
  },
  resolve: {
    modules: ['node_modules'],
  },
  cache: { type: 'filesystem' },
  devtool: debug ? 'source-map' : false,
  target: 'web',
  stats: {
    colors: true,
    reasons: true,
  },
  module: {
    rules: [
      {
        test: /\.(js|jsx)$/,
        exclude: /node_modules/,
        use: {
          loader: 'babel-loader',
          options: { cacheDirectory: true },
        },
      },
      {
        test: /\.(scss|sass)$/,
        use: [
          { loader: 'style-loader' },
          {
            loader: 'css-loader',
            options: { modules: true },
          },
          { loader: 'sass-loader' },
        ],
      },
      {
        test: /\.css$/,
        use: [
          { loader: 'style-loader' },
          { loader: 'css-loader' },
        ],
      },
      {
        test: /\.(ttf|otf|eot|svg|woff(2)?)(\?[a-z0-9]+)?$/,
        type: 'asset/resource',
        generator: {
          filename: 'fonts/[name][ext]',
        },
      },
      {
        test: /\.(png|jpg|gif)$/,
        type: 'asset',
        parser: {
          dataUrlCondition: { maxSize: 8192 },
        },
      },
    ],
  },
  optimization: {
    splitChunks: {
      cacheGroups: {
        commons: {
          test: /[\\/]node_modules[\\/]/,
          name: 'vendors',
          chunks: 'all',
        },
      },
    },
  },
  plugins: debug ? [] : [
    new CompressionWebpackPlugin({
      filename: '[path][base].gz[query]',
      algorithm: 'gzip',
      test: /\.(js|css)$/,
    }),
  ],
};
