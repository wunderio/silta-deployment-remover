'use strict';

// https://github.com/nlf/node-github-hook
var githubhook = require('githubhook');
var github = githubhook({'port': '80', 'path': '/webhooks', 'secret': process.env.WEBHOOKS_SECRET});

var kue = require('kue');
var queue = kue.createQueue({
  redis: `redis://${process.env.REDIS_ADDR}`
});

github.listen();

// github.on('*', function (event, repo, ref, data) {
//   console.log(`Event * | ` + event + ' | ' + repo + ' | ' + ref);
// });

// https://developer.github.com/webhooks/#events
github.on('push', function (project, branch, data) {
  
  if (typeof data.deleted !== 'undefined' && typeof data.after !== 'undefined') {
    
    // Special commit state for when the branch was removed
    if ((data.deleted == true) && (data.after == '0000000000000000000000000000000000000000')) {
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
  }
});
