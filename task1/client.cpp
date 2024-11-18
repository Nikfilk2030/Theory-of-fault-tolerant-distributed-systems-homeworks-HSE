#include "utils.h"

struct TInputParams {
    TTask task;

    TInputParams(int argc, char* argv[]) {
        try {
            if (argc != 4) {
                throw std::runtime_error("ERROR Wrong number of arguments");
            }
            task = TTask(std::stod(argv[1]), std::stod(argv[2]), std::stod(argv[3]));
        } catch (const std::exception& e) {
            std::cerr << e.what() << std::endl;
            exit(1);
        }
    }
};

int main(int argc, char* argv[]) {
    TInputParams params(argc, argv);

    log_client("starting " + params.task.as_string());

    std::vector<TTask> tasks;

    for (double x = params.task.get_start(); x < params.task.get_end(); x += 1.0) {
        TTask task;
        task.set_start(x);
        task.set_end(std::min(x + 1.0, params.task.get_end()));
        task.set_step(params.task.get_step());
        tasks.push_back(task);
    }

    log_client("Начало работы мастера");

    discoverServers();

    double result = distribute_tasks(tasks);

    log_client("Итоговый результат: " + std::to_string(result));
}
