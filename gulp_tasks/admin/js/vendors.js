'use strict';

function Task(lib, gulp) {
  let plugins = lib.plugins,
    prefix = lib.pathto('javascripts/vendors') + '/';

  gulp.task('vendors.js', function () {
    return gulp
      // '!' + prefix + '/jquery.min.js',
      .src([prefix + 'jquery.min.js', prefix + '*.js'])
      .pipe(plugins.concat('vendors.js'))
      .pipe(gulp.dest(lib.pathto('javascripts')));
  });
}

exports.Task = Task;
