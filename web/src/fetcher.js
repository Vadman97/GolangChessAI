import fetch from 'isomorphic-fetch';

function request(url, options) {
  const defaultOptions = {
    credentials: 'same-origin',
    headers: {
      'Accept': 'application/json',
      'Content-Type': 'application/json',
      'Content-Security-Policy': 'default-src \'self\'',
      'X-Frame-Options': 'SAMEORIGIN',
      'X-XSS-Protection': 1,
    },
  };

  if (options.body && typeof options.body === 'object') {
    options.body = JSON.stringify(options.body);
  }

  const combinedOptions = { ...defaultOptions, ...options };

  return new Promise((resolve, reject) => {
    let responseOk = false;
    fetch(url, combinedOptions)
    .then(res => {
      responseOk = res.ok;
      return res.json();
    })
    .then(body => {
      if (responseOk) {
        resolve(body);
      }
      throw body;
    })
    .catch(error => {
      reject(error);
    })
  });
}

const fetcher = {};
['get', 'post', 'put', 'delete'].forEach((method) => {
  fetcher[method] = (url, options = {}) => {
    options.method = method;
    return request(url, options);
  };
});

export default fetcher;
