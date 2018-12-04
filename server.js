'use strict';

var kue = require('kue');
var queue = kue.createQueue({
  redis: {
    port: 6379,
    host: process.env.REDIS_HOST,
    auth: process.env.REDIS_PASSWORD
  }
});

// https://github.com/nlf/node-github-hook
var githubhook = require('githubhook');
var github = githubhook({
  'port': '80', 
  'path': '/webhooks', 
  'secret': process.env.WEBHOOKS_SECRET 
});
github.listen();

// github.on('*', function (event, project, branch, data) {
//   console.log('SERVER: Webhook event * |', event, '|', project, '|', branch);
// });

// https://developer.github.com/webhooks/#events
github.on('push', function (project, branch, data) {
  if (typeof data.deleted !== 'undefined' && typeof data.after !== 'undefined') {
    // Special commit state for when the branch was removed
    if ((data.deleted == true) && (data.after == '0000000000000000000000000000000000000000')) {      
      queue_branch_removal(project, branch, data)
    }
  }
});

// https://developer.github.com/webhooks/#events
github.on('delete', function (project, branch, data) {
  queue_branch_removal(project, branch, data)
});

// Adds branch removal job to remover queue
function queue_branch_removal(project, branch, data) {
  console.log('SERVER: Branch deletion event');

  var job = queue.create('remover', {
    url: data.repository.url,
    project: project,
    branch: branch
  });

  job
    .on('enqueue', function (){
      console.log('SERVER: Job', job.id, 'created');
    })
    .on('failed', function (errorMessage){
      console.log('SERVER: Job', job.id, 'has failed:', errorMessage);
    })
    .removeOnComplete(true)
    .attempts(5)
    .save(function(err) {
      if (err) {
        console.error(err);
      }
    });
}
