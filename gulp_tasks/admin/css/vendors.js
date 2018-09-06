'use strict';

function Task(lib, gulp) {
  var plugins = lib.plugins,
    prefix = lib.pathto('stylesheets/vendors') + '/';

  gulp.task('vendors.css', function () {
    return gulp
      .src([prefix + '*.css'])
      .pipe(plugins.concat('vendors.css'))
      .pipe(gulp.dest(lib.pathto('stylesheets')));
  });
}

exports.Task = Task;
