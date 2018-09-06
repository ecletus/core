'use strict';

function Task(lib, gulp) {
  var styles = lib.styles,
    plugins = lib.plugins;

  gulp.task('release.css', function () {
    return gulp
      .src(styles.qorAdmin)
      .pipe(plugins.concat('qor_admin_default.css'))
      .pipe(gulp.dest(styles.dest));
  });

  gulp.task('release.css+', ['css', 'release.css']);
}

exports.Task = Task;
