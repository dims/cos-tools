# How to contribute

We'd love to accept your patches and contributions to this project. There are
just a few small guidelines you need to follow.

## Contributor License Agreement

Contributions to this project must be accompanied by a Contributor License
Agreement (CLA). You (or your employer) retain the copyright to your
contribution; this simply gives us permission to use and redistribute your
contributions as part of the project. Head over to
<https://cla.developers.google.com/> to see your current agreements on file or
to sign a new one.

You generally only need to submit a CLA once, so if you've already submitted one
(even if it was for a different project), you probably don't need to do it
again.

## Checking out source

### Set up git

If you haven't already, configure your git user email and name:

```
git config --global user.email "<your-email>@example.com"
git config --global user.name "Your Name"
```

### Checkout

With your git configured, you can now checkout the source with:

```
git clone https://cos.googlesource.com/cos/tools && (cd tools && f=`git rev-parse --git-dir`/hooks/commit-msg ; mkdir -p $(dirname $f) ; curl -Lo $f https://gerrit-review.googlesource.com/tools/hooks/commit-msg ; chmod +x $f)
```

The last command not only clones the repository but also adds a git pre-commit
hook that will be explained below in "Making changes".

## Making changes

The typical git workflow applies here. You can use `git checkout -b` to create a
branch and `git commit` to commit local changes. The pre-commit hook that was
downloaded earlier will insert a Gerrit (Gerrit is explained below) Change-Id
into every local commit you make. This Change-Id is specific to Gerrit and
defines a change: one change and the corresponding review in Gerrit will have
one Change-Id. If a Change-Id is not provided in the commit message, Gerrit will
reject the commit. While you can manually add a Change-Id later, it's strongly
recommended you include it from the start, when you clone the repository.

## Code reviews

All submissions, including submissions by project members, require review. We
use [Gerrit](https://gerrit-review.googlesource.com/Documentation/) for this
purpose.

### Register with Gerrit

Before you can request code reviews, you need to register with Gerrit first. You
can do that by going to <https://cos-review.googlesource.com> and clicking "Sign
in" at the top right corner.

### Set up git cookies

You will also need to generate and configure your git cookies in order to push
changes to Gerrit. The following steps help you with that:

1.  Go to https://cos-review.googlesource.com/new-password
1.  Log in with your google account
1.  Follow the on-screen directions page to set up/append to your
    `~/.gitcookies` file
1.  Verify that your cookies are correctly setup:

    ```
    git ls-remote https://cos.googlesource.com/cos/tools.git
    ```

### Upload a CL

```
git push origin HEAD:refs/for/master
```

The command will print a URL to the CL that has just been uploaded. You can
follow the URL to the Gerrit UI and add reviewers from there.

## Community guidelines

This project follows
[Google's Open Source Community Guidelines](https://opensource.google/conduct/).
