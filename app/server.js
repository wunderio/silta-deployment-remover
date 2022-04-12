'use strict';

const crypto = require('crypto');
const child_process = require('child_process');

const kue = require('kue');
console.log('Using redis host:', process.env.REDIS_HOST);
const queue = kue.createQueue({
  redis: {
    port: 6379,
    host: process.env.REDIS_HOST,
    auth: process.env.REDIS_PASSWORD
  }
});

// https://github.com/wunderio/node-github-hook
const githubhook = require('githubhook');
const github = githubhook({
  'port': '80', 
  'path': '/webhooks',
  'secret': process.env.WEBHOOKS_SECRET 
});
github.listen();

// Webhook - delete event
github.on('delete', function (project, branch, data) {
  // https://developer.github.com/webhooks/#events
  queue_branch_removal(project, branch, data)
});

// Webhook - push event
github.on('push', function (project, branch, data) {
  if (typeof data.deleted !== 'undefined' && typeof data.after !== 'undefined') {
    // Special commit state for when the branch was removed
    // https://developer.github.com/webhooks/#events
    if ((data.deleted == true) && (data.after == '0000000000000000000000000000000000000000')) {      
      branch = branch.replace(/^(refs\/heads\/)/,'');
      branch = branch.replace(/^(refs\/)/,'');
      queue_branch_removal(project, branch, data)
    }
  }
});

queue.process('remover', function (job, done){
  console.log('REMOVER [', job.id, ']: Job started');

  // Pass repo name as environment variable for the namespace.
  process.env.NAMESPACE = job.data.project.toLowerCase().replace(/[^a-z0-9]/gi,'-');;

  // Pass release name as environment variable (legacy mode)
  process.env.RELEASE_NAME = job.data.branch.toLowerCase().replace(/[^a-z0-9]/gi,'-');

  // Pass branch name as environment variable
  process.env.BRANCH_NAME = job.data.branch;
  
  console.log('REMOVER [', job.id, ']: Branchname', job.data.branch);

  // Log on to cluster and remove helm deployment
  child_process.exec('/app/delete-deployment.sh', function (error, stdout, stderr) {
    console.log(stdout);
    if (error) {
      console.log('ERROR:', stderr);
      done(error);
    }
    else {
      done();
    }
  });
});

// Adds branch removal job to remover queue
function queue_branch_removal(project, branch, data) {
  console.log('SERVER: Branch deletion event');

  // Create new remover job
  const job = queue.create('remover', {
    url: data.repository.url,
    project: project,
    branch: branch
  });

  // Attach event listeners
  job
    .on('enqueue', function (){
      console.log('SERVER: Job', job.id, 'created');
    })
    .on('complete', function (){
      console.log('SERVER: Job', job.id, 'completed');
    })
    .on('failed', function (errorMessage){
      console.log('SERVER: Job', job.id, 'has failed:', errorMessage);
    })
  
  // Save job to queue
  job.removeOnComplete(true)
    .attempts(5)
    .save(function(err) {
      if (err) {
        console.error(err);
      }
    });
}
