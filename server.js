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

// https://github.com/nlf/node-github-hook
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
      queue_branch_removal(project, branch, data)
    }
  }
});

queue.process('remover', function (job, done){
  console.log('REMOVER: Job', job.id, 'started');

  // Calculate release name to reflect this one
  // https://github.com/wunderio/silta-circleci/blob/feature/add-deployproc-scripts/utils/set_release_name.sh
  // TODO: Select release name with deployment "branchname" label.
  // TODO: This could be used in future too: https://github.com/helm/helm/issues/4639
  let branchname = job.data.branch.toLowerCase().replace(/[^a-z0-9]/gi,'-');
  const branchname_hash = crypto.createHash('sha256').update(branchname).digest("hex").substring(0, 4);
  const branchname_truncated = branchname.substring(0, 15).replace(/\-$/, '');
    if (branchname.length >= 20) {
    branchname = branchname_truncated + '-' + branchname_hash;
  }

  let reponame = job.data.project.toLowerCase().replace(/[^a-z0-9]/gi,'-');
  const reponame_hash = crypto.createHash('sha256').update(reponame).digest("hex").substring(0, 4);
  const reponame_truncated = reponame.substring(0, 15).replace(/\-$/, '');

  // Pass unshortened repo name as environment variable for the namespace.
  process.env.NAMESPACE = reponame;

  if (reponame.length >= 20) {
    reponame = reponame_truncated + '-' + reponame_hash;
  }

  const release_name = reponame + '--' + branchname;
  
  // Pass release name as environment variable
  process.env.RELEASE_NAME = release_name;

  // Get the simpler helm3 release name.
  process.env.HELM3_RELEASE_NAME = job.data.branch.toLowerCase().replace(/[^a-z0-9]/gi,'-');
  
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
