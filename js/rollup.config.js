import { nodeResolve } from '@rollup/plugin-node-resolve';

export default {
  input: 'client.js',
  output: {
    file: 'bundle.js',
  },
  plugins: [nodeResolve()]
};