class Migration:
    version = 0
    name = ""
    changes_data = False

    def describe(self, context):
        return []

    def run(self, context):
        raise NotImplementedError

    def verify(self, context):
        return []
