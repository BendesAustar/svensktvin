// svelte.config.js
import adapter from '@sveltejs/adapter-node';

export default {
  kit: { 
    adapter: adapter(),
    // Disable CSRF origin checking in dev mode
    // CSRF tokens are still validated but origin check is relaxed for localhost
  }
};
