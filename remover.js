'use strict';

var exec = require('child_process').exec;
var crypto = require('crypto');

var kue = require('kue');
var REDIS_ADDR = process.env.REDIS_ADDR || '127.0.0.1:6379';
var queue = kue.createQueue({
  redis: `redis://${REDIS_ADDR}`
});

console.log('REMOVER: Waiting for queue entries')

queue.process('remover', function (job, done){
  console.log('REMOVER: Job', job.id, 'started');

  // Calculate release name to reflect this one
  // https://github.com/wunderio/silta-circleci/blob/feature/add-deployproc-scripts/utils/set_release_name.sh
  // TODO: Select release name with deployment "branchname" label.
  var branchname = job.data.branch.toLowerCase().replace(/[^a-z0-9]/gi,'');
  var branchname_hash = crypto.createHash('sha256').update(branchname).digest("hex").substring(1, 4);
  var branchname_truncated = branchname.substring(1, 15).replace(/\-$/, '');
	var reponame = job.data.project.toLowerCase().replace(/[^a-z0-9]/gi,'');
  var reponame_hash = crypto.createHash('sha256').update(reponame).digest("hex").substring(1, 4);
  var reponame_truncated = reponame.substring(1, 15).replace(/\-$/, '');
  
  if (branchname.length >= 25) {
  	branchname = branchname_truncated + '-' + branchname_hash;
  }
  if (reponame.length >= 25) {
  	reponame = reponame_truncated + '-' + reponame_hash;
  }

  var release_name = reponame + '--' + branchname;
  
  // Pass release name as environment variable
  process.env.RELEASE_NAME = release_name;
  
  exec('/app/delete-deployment.sh', function(error, stdout, stderr) {
  	console.log(stdout);
  	console.log(stderr);
  	if (error !== null) {
		console.log('exec error: ' + error);
  	}

  	console.log('SERVER: Job', job.id, 'completed');
  });

  done();
});
